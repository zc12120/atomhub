package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"io"
	"path/filepath"
	"testing"
	"time"

	"github.com/zc12120/atomhub/internal/config"
	"github.com/zc12120/atomhub/internal/types"
)

func testConfig(t *testing.T) config.Config {
	t.Helper()
	return config.Config{
		HTTPAddr:      ":0",
		DBPath:        filepath.Join(t.TempDir(), "atomhub.db"),
		SessionSecret: "test-session-secret",
		SessionTTL:    time.Hour,
		AdminUsername: "admin",
		AdminPassword: "admin",
		GatewayToken:  "gateway-token",
	}
}

func TestAdminLoginAndSession(t *testing.T) {
	app, err := New(testConfig(t))
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	defer app.Close()

	body := []byte(`{"username":"admin","password":"admin"}`)
	req := httptest.NewRequest(http.MethodPost, "/admin/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	app.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("login status = %d body=%s", rec.Code, rec.Body.String())
	}

	sessionReq := httptest.NewRequest(http.MethodGet, "/admin/session", nil)
	for _, cookie := range rec.Result().Cookies() {
		sessionReq.AddCookie(cookie)
	}
	sessionRec := httptest.NewRecorder()
	app.Handler.ServeHTTP(sessionRec, sessionReq)
	if sessionRec.Code != http.StatusOK {
		t.Fatalf("session status = %d body=%s", sessionRec.Code, sessionRec.Body.String())
	}
	var payload adminSessionResponse
	if err := json.Unmarshal(sessionRec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode session payload: %v", err)
	}
	if !payload.Authenticated || payload.Username != "admin" {
		t.Fatalf("unexpected session payload: %+v", payload)
	}
}

func TestGatewayModelsAndProxyOpenAI(t *testing.T) {
	app, err := New(testConfig(t))
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	defer app.Close()
	app.upstreamClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/v1/chat/completions" {
			return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(bytes.NewReader([]byte(`{"error":"not found"}`))), Header: make(http.Header)}, nil
		}
		body := []byte(`{"id":"chatcmpl-test","object":"chat.completion","created":1,"model":"gpt-4o-mini","choices":[{"index":0,"message":{"role":"assistant","content":"hello back"},"finish_reason":"stop"}],"usage":{"prompt_tokens":5,"completion_tokens":3,"total_tokens":8}}`)
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(body)), Header: http.Header{"Content-Type": []string{"application/json"}}}, nil
	})}

	key, err := app.keyStore.Create(context.Background(), types.UpstreamKey{Name: "openai-key", Provider: types.ProviderOpenAI, BaseURL: "https://example.com", APIKey: "sk-test", Enabled: true})
	if err != nil {
		t.Fatalf("create key: %v", err)
	}
	if err := app.stateStore.Ensure(context.Background(), key.ID); err != nil {
		t.Fatalf("ensure state: %v", err)
	}
	if err := app.modelStore.ReplaceForKey(context.Background(), key.ID, []string{"gpt-4o-mini"}); err != nil {
		t.Fatalf("replace key models: %v", err)
	}
	if err := app.catalog.Rebuild(context.Background()); err != nil {
		t.Fatalf("rebuild catalog: %v", err)
	}

	modelsReq := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	modelsReq.Header.Set("Authorization", "Bearer gateway-token")
	modelsRec := httptest.NewRecorder()
	app.Handler.ServeHTTP(modelsRec, modelsReq)
	if modelsRec.Code != http.StatusOK {
		t.Fatalf("models status = %d body=%s", modelsRec.Code, modelsRec.Body.String())
	}
	if !bytes.Contains(modelsRec.Body.Bytes(), []byte("gpt-4o-mini")) {
		t.Fatalf("expected model listing to contain gpt-4o-mini: %s", modelsRec.Body.String())
	}

	chatBody := []byte(`{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hi"}]}`)
	chatReq := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(chatBody))
	chatReq.Header.Set("Authorization", "Bearer gateway-token")
	chatRec := httptest.NewRecorder()
	app.Handler.ServeHTTP(chatRec, chatReq)
	if chatRec.Code != http.StatusOK {
		t.Fatalf("chat status = %d body=%s", chatRec.Code, chatRec.Body.String())
	}
	if !bytes.Contains(chatRec.Body.Bytes(), []byte("hello back")) {
		t.Fatalf("expected proxied response: %s", chatRec.Body.String())
	}
	logs, err := app.logStore.ListRecent(context.Background(), 10)
	if err != nil {
		t.Fatalf("list logs: %v", err)
	}
	if len(logs) != 1 || logs[0].TotalTokens != 8 {
		t.Fatalf("unexpected logs: %#v", logs)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }
