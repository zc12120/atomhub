package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zc12120/atomhub/internal/types"
)

func TestAdminRequestsReturnsRecentLogsAndSummary(t *testing.T) {
	app, err := New(testConfig(t))
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	defer app.Close()

	key, err := app.keyStore.Create(context.Background(), types.UpstreamKey{
		Name:     "primary",
		Provider: types.ProviderOpenAI,
		BaseURL:  "https://api.openai.com",
		APIKey:   "sk-test",
		Enabled:  true,
	})
	if err != nil {
		t.Fatalf("create key: %v", err)
	}
	if err := app.stateStore.Ensure(context.Background(), key.ID); err != nil {
		t.Fatalf("ensure state: %v", err)
	}
	_, _ = app.logStore.Insert(context.Background(), key.ID, "gpt-4o-mini", types.UsageTokens{PromptTokens: 5, CompletionTokens: 2, TotalTokens: 7}, 120*time.Millisecond, nil)
	_, _ = app.logStore.Insert(context.Background(), key.ID, "gpt-4o-mini", types.UsageTokens{PromptTokens: 3, CompletionTokens: 1, TotalTokens: 4}, 90*time.Millisecond, nil)
	_, _ = app.logStore.Insert(context.Background(), key.ID, "claude-3-5-haiku", types.UsageTokens{}, 200*time.Millisecond, context.DeadlineExceeded)

	loginReq := httptest.NewRequest(http.MethodPost, "/admin/login", bytes.NewReader([]byte(`{"username":"admin","password":"admin"}`)))
	loginRec := httptest.NewRecorder()
	app.Handler.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login status = %d body=%s", loginRec.Code, loginRec.Body.String())
	}

	req := httptest.NewRequest(http.MethodGet, "/admin/requests?model=gpt-4o-mini", nil)
	for _, cookie := range loginRec.Result().Cookies() {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	app.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("requests status = %d body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Items []struct {
			Model string `json:"model"`
		} `json:"items"`
		Summary struct {
			RequestCount int64 `json:"request_count"`
			TotalTokens  int64 `json:"total_tokens"`
			ErrorCount   int64 `json:"error_count"`
		} `json:"summary"`
		Filters struct {
			Model  string   `json:"model"`
			Models []string `json:"models"`
		} `json:"filters"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.Summary.RequestCount != 2 || payload.Summary.TotalTokens != 11 || payload.Summary.ErrorCount != 0 {
		t.Fatalf("unexpected summary: %+v", payload.Summary)
	}
	if payload.Filters.Model != "gpt-4o-mini" {
		t.Fatalf("unexpected filter: %+v", payload.Filters)
	}
	if len(payload.Items) != 2 {
		t.Fatalf("unexpected item count: %d", len(payload.Items))
	}
	for _, item := range payload.Items {
		if item.Model != "gpt-4o-mini" {
			t.Fatalf("unexpected item: %+v", item)
		}
	}
}
