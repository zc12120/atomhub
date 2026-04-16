package app

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/zc12120/atomhub/internal/types"
)

func TestGatewayAuthAcceptsDownstreamKeyAndAttributesUsage(t *testing.T) {
	app, err := New(testConfig(t))
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	defer app.Close()

	app.upstreamClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body := []byte(`{"id":"chatcmpl-test","object":"chat.completion","created":1,"model":"gpt-4o-mini","choices":[{"index":0,"message":{"role":"assistant","content":"hello back"},"finish_reason":"stop"}],"usage":{"prompt_tokens":5,"completion_tokens":3,"total_tokens":8}}`)
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(body)), Header: http.Header{"Content-Type": []string{"application/json"}}}, nil
	})}

	seedKeyWithModel(t, app, types.UpstreamKey{Name: "openai-key", Provider: types.ProviderOpenAI, BaseURL: "https://example.com", APIKey: "sk-test", Enabled: true}, "gpt-4o-mini")

	downstreamKey, plaintextToken, err := app.downstreamKeyStore.Create(context.Background(), types.DownstreamKey{Name: "client-a", Enabled: true})
	if err != nil {
		t.Fatalf("create downstream key: %v", err)
	}

	chatBody := []byte(`{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hi"}]}`)
	chatReq := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(chatBody))
	chatReq.Header.Set("Authorization", "Bearer "+plaintextToken)
	chatRec := httptest.NewRecorder()
	app.Handler.ServeHTTP(chatRec, chatReq)
	if chatRec.Code != http.StatusOK {
		t.Fatalf("chat status = %d body=%s", chatRec.Code, chatRec.Body.String())
	}

	logs, err := app.logStore.ListRecent(context.Background(), 10)
	if err != nil {
		t.Fatalf("list logs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}
	if logs[0].DownstreamKeyID == nil || *logs[0].DownstreamKeyID != downstreamKey.ID {
		t.Fatalf("expected downstream key attribution, got %+v", logs[0])
	}

	storedKey, err := app.downstreamKeyStore.Get(context.Background(), downstreamKey.ID)
	if err != nil {
		t.Fatalf("get downstream key: %v", err)
	}
	if storedKey.RequestCount != 1 || storedKey.TotalTokens != 8 || storedKey.PromptTokens != 5 || storedKey.CompletionTokens != 3 {
		t.Fatalf("unexpected downstream key usage: %+v", storedKey)
	}
	if storedKey.LastUsedAt == nil {
		t.Fatalf("expected last_used_at to be set: %+v", storedKey)
	}
}

func TestGatewayAuthRejectsDisabledDownstreamKey(t *testing.T) {
	app, err := New(testConfig(t))
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	defer app.Close()

	created, plaintextToken, err := app.downstreamKeyStore.Create(context.Background(), types.DownstreamKey{Name: "client-a", Enabled: true})
	if err != nil {
		t.Fatalf("create downstream key: %v", err)
	}
	updated, err := app.downstreamKeyStore.Update(context.Background(), types.DownstreamKey{ID: created.ID, Name: created.Name, Enabled: false})
	if err != nil {
		t.Fatalf("disable downstream key: %v", err)
	}
	if updated.Enabled {
		t.Fatalf("expected disabled downstream key: %+v", updated)
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	req.Header.Set("Authorization", "Bearer "+plaintextToken)
	rec := httptest.NewRecorder()
	app.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGatewayAuthEnvFallbackStillWorksWithoutDownstreamKey(t *testing.T) {
	app, err := New(testConfig(t))
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	defer app.Close()

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	req.Header.Set("Authorization", "Bearer gateway-token")
	rec := httptest.NewRecorder()
	app.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected env fallback token to work, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAdminDownstreamKeyCRUD(t *testing.T) {
	app, err := New(testConfig(t))
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	defer app.Close()

	cookies := loginAdmin(t, app)

	createReq := httptest.NewRequest(http.MethodPost, "/admin/downstream-keys", bytes.NewReader([]byte(`{"name":"client-a"}`)))
	for _, cookie := range cookies {
		createReq.AddCookie(cookie)
	}
	createRec := httptest.NewRecorder()
	app.Handler.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create status = %d body=%s", createRec.Code, createRec.Body.String())
	}
	var created adminDownstreamKeyCreateResponse
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create payload: %v", err)
	}
	if created.Token == "" || created.Item.ID == 0 || created.Item.Name != "client-a" || !created.Item.Enabled {
		t.Fatalf("unexpected create payload: %+v", created)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/admin/downstream-keys", nil)
	for _, cookie := range cookies {
		listReq.AddCookie(cookie)
	}
	listRec := httptest.NewRecorder()
	app.Handler.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d body=%s", listRec.Code, listRec.Body.String())
	}
	var listed adminDownstreamKeysResponse
	if err := json.Unmarshal(listRec.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode list payload: %v", err)
	}
	if len(listed.Items) != 1 || listed.Items[0].ID != created.Item.ID {
		t.Fatalf("unexpected list payload: %+v", listed)
	}
	if listed.Items[0].MaskedToken == "" || !listed.Items[0].CanReveal {
		t.Fatalf("expected masked token and reveal support: %+v", listed.Items[0])
	}

	revealReq := httptest.NewRequest(http.MethodGet, "/admin/downstream-keys/"+itoa(created.Item.ID)+"/token", nil)
	for _, cookie := range cookies {
		revealReq.AddCookie(cookie)
	}
	revealRec := httptest.NewRecorder()
	app.Handler.ServeHTTP(revealRec, revealReq)
	if revealRec.Code != http.StatusOK {
		t.Fatalf("reveal status = %d body=%s", revealRec.Code, revealRec.Body.String())
	}
	var revealed adminDownstreamKeyTokenResponse
	if err := json.Unmarshal(revealRec.Body.Bytes(), &revealed); err != nil {
		t.Fatalf("decode reveal payload: %v", err)
	}
	if revealed.Token != created.Token || revealed.ID != created.Item.ID {
		t.Fatalf("unexpected reveal payload: %+v created=%+v", revealed, created)
	}

	regenerateReq := httptest.NewRequest(http.MethodPost, "/admin/downstream-keys/"+itoa(created.Item.ID)+"/regenerate", nil)
	for _, cookie := range cookies {
		regenerateReq.AddCookie(cookie)
	}
	regenerateRec := httptest.NewRecorder()
	app.Handler.ServeHTTP(regenerateRec, regenerateReq)
	if regenerateRec.Code != http.StatusOK {
		t.Fatalf("regenerate status = %d body=%s", regenerateRec.Code, regenerateRec.Body.String())
	}
	var regenerated adminDownstreamKeyTokenResponse
	if err := json.Unmarshal(regenerateRec.Body.Bytes(), &regenerated); err != nil {
		t.Fatalf("decode regenerate payload: %v", err)
	}
	if regenerated.Token == "" || regenerated.Token == created.Token || regenerated.ID != created.Item.ID {
		t.Fatalf("unexpected regenerate payload: %+v created=%+v", regenerated, created)
	}

	updateReq := httptest.NewRequest(http.MethodPut, "/admin/downstream-keys/"+itoa(created.Item.ID), bytes.NewReader([]byte(`{"name":"client-b","enabled":false}`)))
	for _, cookie := range cookies {
		updateReq.AddCookie(cookie)
	}
	updateRec := httptest.NewRecorder()
	app.Handler.ServeHTTP(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("update status = %d body=%s", updateRec.Code, updateRec.Body.String())
	}
	var updated adminDownstreamKeyItem
	if err := json.Unmarshal(updateRec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode update payload: %v", err)
	}
	if updated.Name != "client-b" || updated.Enabled {
		t.Fatalf("unexpected updated payload: %+v", updated)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/admin/downstream-keys/"+itoa(created.Item.ID), nil)
	for _, cookie := range cookies {
		deleteReq.AddCookie(cookie)
	}
	deleteRec := httptest.NewRecorder()
	app.Handler.ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d body=%s", deleteRec.Code, deleteRec.Body.String())
	}

	listAfterReq := httptest.NewRequest(http.MethodGet, "/admin/downstream-keys", nil)
	for _, cookie := range cookies {
		listAfterReq.AddCookie(cookie)
	}
	listAfterRec := httptest.NewRecorder()
	app.Handler.ServeHTTP(listAfterRec, listAfterReq)
	if listAfterRec.Code != http.StatusOK {
		t.Fatalf("list after delete status = %d body=%s", listAfterRec.Code, listAfterRec.Body.String())
	}
	var listedAfter adminDownstreamKeysResponse
	if err := json.Unmarshal(listAfterRec.Body.Bytes(), &listedAfter); err != nil {
		t.Fatalf("decode list after payload: %v", err)
	}
	if len(listedAfter.Items) != 0 {
		t.Fatalf("expected empty list after delete: %+v", listedAfter)
	}
}

func loginAdmin(t *testing.T, app *App) []*http.Cookie {
	t.Helper()
	loginReq := httptest.NewRequest(http.MethodPost, "/admin/login", bytes.NewReader([]byte(`{"username":"admin","password":"admin"}`)))
	loginRec := httptest.NewRecorder()
	app.Handler.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login status = %d body=%s", loginRec.Code, loginRec.Body.String())
	}
	return loginRec.Result().Cookies()
}

func itoa(v int64) string {
	return strconv.FormatInt(v, 10)
}
