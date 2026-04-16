package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zc12120/atomhub/internal/types"
)

func TestAdminUpdateKeyAllowsPartialPayload(t *testing.T) {
	app, err := New(testConfig(t))
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	defer app.Close()

	key, err := app.keyStore.Create(context.Background(), types.UpstreamKey{
		Name:     "primary",
		Provider: types.ProviderOpenAI,
		BaseURL:  "https://api.openai.com",
		APIKey:   "sk-old",
		Enabled:  true,
	})
	if err != nil {
		t.Fatalf("create key: %v", err)
	}
	if err := app.stateStore.Ensure(context.Background(), key.ID); err != nil {
		t.Fatalf("ensure state: %v", err)
	}

	loginReq := httptest.NewRequest(http.MethodPost, "/admin/login", bytes.NewReader([]byte(`{"username":"admin","password":"admin"}`)))
	loginRec := httptest.NewRecorder()
	app.Handler.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login status = %d body=%s", loginRec.Code, loginRec.Body.String())
	}

	updateReq := httptest.NewRequest(http.MethodPut, "/admin/keys/1", bytes.NewReader([]byte(`{"enabled":false}`)))
	for _, cookie := range loginRec.Result().Cookies() {
		updateReq.AddCookie(cookie)
	}
	updateRec := httptest.NewRecorder()
	app.Handler.ServeHTTP(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("update status = %d body=%s", updateRec.Code, updateRec.Body.String())
	}

	var updated adminKeyItem
	if err := json.Unmarshal(updateRec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode update payload: %v", err)
	}
	if updated.Enabled {
		t.Fatalf("expected key to be disabled: %+v", updated)
	}
	if updated.Label != "primary" || updated.Provider != string(types.ProviderOpenAI) || updated.BaseURL != "https://api.openai.com" {
		t.Fatalf("expected other fields unchanged: %+v", updated)
	}
	stored, err := app.keyStore.Get(context.Background(), key.ID)
	if err != nil {
		t.Fatalf("get key: %v", err)
	}
	if stored.Enabled {
		t.Fatalf("expected stored key to be disabled: %+v", stored)
	}
	if stored.APIKey != "sk-old" {
		t.Fatalf("expected api key unchanged: %+v", stored)
	}
}
