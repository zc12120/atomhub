package app

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zc12120/atomhub/internal/config"
	"github.com/zc12120/atomhub/internal/types"
)

func testConfig(t *testing.T) config.Config {
	t.Helper()
	return config.Config{
		HTTPAddr:            ":0",
		DBPath:              filepath.Join(t.TempDir(), "atomhub.db"),
		SessionSecret:       "test-session-secret",
		SessionTTL:          time.Hour,
		AdminUsername:       "admin",
		AdminPassword:       "admin",
		GatewayToken:        "gateway-token",
		DownstreamKeySecret: "test-downstream-secret",
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

func TestChatCompletionsStreamOpenAI(t *testing.T) {
	app, err := New(testConfig(t))
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	defer app.Close()

	var sawStreamRequest bool
	app.upstreamClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/v1/chat/completions" {
			return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader(`{"error":"not found"}`)), Header: make(http.Header)}, nil
		}
		requestBody, _ := io.ReadAll(req.Body)
		sawStreamRequest = bytes.Contains(requestBody, []byte(`"stream":true`))

		body := strings.Join([]string{
			`data: {"id":"chatcmpl-stream","object":"chat.completion.chunk","created":1,"model":"gpt-4o-mini","choices":[{"index":0,"delta":{"role":"assistant"}]}`,
			``,
			`data: {"id":"chatcmpl-stream","object":"chat.completion.chunk","created":1,"model":"gpt-4o-mini","choices":[{"index":0,"delta":{"content":"hello stream"}}]}`,
			``,
			`data: {"id":"chatcmpl-stream","object":"chat.completion.chunk","created":1,"model":"gpt-4o-mini","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":2,"completion_tokens":1,"total_tokens":3}}`,
			``,
			`data: [DONE]`,
			``,
		}, "\n")
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		}, nil
	})}

	seedKeyWithModel(t, app, types.UpstreamKey{Name: "openai-stream-key", Provider: types.ProviderOpenAI, BaseURL: "https://example.com", APIKey: "sk-test", Enabled: true}, "gpt-4o-mini")

	chatBody := []byte(`{"model":"gpt-4o-mini","stream":true,"messages":[{"role":"user","content":"hi"}]}`)
	chatReq := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(chatBody))
	chatReq.Header.Set("Authorization", "Bearer gateway-token")
	chatRec := httptest.NewRecorder()
	app.Handler.ServeHTTP(chatRec, chatReq)

	if chatRec.Code != http.StatusOK {
		t.Fatalf("stream chat status = %d body=%s", chatRec.Code, chatRec.Body.String())
	}
	if !strings.HasPrefix(chatRec.Header().Get("Content-Type"), "text/event-stream") {
		t.Fatalf("expected text/event-stream content type, got %q", chatRec.Header().Get("Content-Type"))
	}
	body := chatRec.Body.String()
	if !strings.Contains(body, "hello stream") {
		t.Fatalf("expected streamed content, got %s", body)
	}
	if !strings.Contains(body, "data: [DONE]") {
		t.Fatalf("expected done marker, got %s", body)
	}
	if !sawStreamRequest {
		t.Fatalf("expected upstream request body to include stream=true")
	}

	logs, err := app.logStore.ListRecent(context.Background(), 10)
	if err != nil {
		t.Fatalf("list logs: %v", err)
	}
	if len(logs) != 1 || logs[0].TotalTokens != 3 {
		t.Fatalf("unexpected logs: %#v", logs)
	}
}

func TestChatCompletionsStreamAnthropicFallback(t *testing.T) {
	app, err := New(testConfig(t))
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	defer app.Close()

	app.upstreamClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/v1/messages" {
			return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader(`{"error":"not found"}`)), Header: make(http.Header)}, nil
		}
		body := `{"id":"msg-test","model":"claude-3-5-haiku","usage":{"input_tokens":6,"output_tokens":4},"content":[{"type":"text","text":"hello from claude"}],"stop_reason":"end_turn"}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}, nil
	})}

	seedKeyWithModel(t, app, types.UpstreamKey{Name: "anthropic-stream-key", Provider: types.ProviderAnthropic, BaseURL: "https://example.com", APIKey: "anthropic-test", Enabled: true}, "claude-3-5-haiku")

	chatBody := []byte(`{"model":"claude-3-5-haiku","stream":true,"messages":[{"role":"user","content":"hi"}]}`)
	chatReq := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(chatBody))
	chatReq.Header.Set("Authorization", "Bearer gateway-token")
	chatRec := httptest.NewRecorder()
	app.Handler.ServeHTTP(chatRec, chatReq)

	if chatRec.Code != http.StatusOK {
		t.Fatalf("stream chat status = %d body=%s", chatRec.Code, chatRec.Body.String())
	}
	if !strings.HasPrefix(chatRec.Header().Get("Content-Type"), "text/event-stream") {
		t.Fatalf("expected text/event-stream content type, got %q", chatRec.Header().Get("Content-Type"))
	}
	body := chatRec.Body.String()
	if !strings.Contains(body, "chat.completion.chunk") || !strings.Contains(body, "hello from claude") {
		t.Fatalf("expected synthesized stream chunks, got %s", body)
	}
	if !strings.Contains(body, "data: [DONE]") {
		t.Fatalf("expected done marker, got %s", body)
	}

	logs, err := app.logStore.ListRecent(context.Background(), 10)
	if err != nil {
		t.Fatalf("list logs: %v", err)
	}
	if len(logs) != 1 || logs[0].TotalTokens != 10 {
		t.Fatalf("unexpected logs: %#v", logs)
	}
}

func TestChatCompletionsStreamGeminiFallback(t *testing.T) {
	app, err := New(testConfig(t))
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	defer app.Close()

	app.upstreamClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if !strings.HasPrefix(req.URL.Path, "/v1beta/models/gemini-2.0-flash:generateContent") {
			return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader(`{"error":"not found"}`)), Header: make(http.Header)}, nil
		}
		body := `{"candidates":[{"finishReason":"STOP","content":{"parts":[{"text":"hello from gemini"}]}}],"usageMetadata":{"promptTokenCount":3,"candidatesTokenCount":2,"totalTokenCount":5}}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}, nil
	})}

	seedKeyWithModel(t, app, types.UpstreamKey{Name: "gemini-stream-key", Provider: types.ProviderGemini, BaseURL: "https://example.com", APIKey: "gemini-test", Enabled: true}, "gemini-2.0-flash")

	chatBody := []byte(`{"model":"gemini-2.0-flash","stream":true,"messages":[{"role":"user","content":"hi"}]}`)
	chatReq := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(chatBody))
	chatReq.Header.Set("Authorization", "Bearer gateway-token")
	chatRec := httptest.NewRecorder()
	app.Handler.ServeHTTP(chatRec, chatReq)

	if chatRec.Code != http.StatusOK {
		t.Fatalf("stream chat status = %d body=%s", chatRec.Code, chatRec.Body.String())
	}
	if !strings.HasPrefix(chatRec.Header().Get("Content-Type"), "text/event-stream") {
		t.Fatalf("expected text/event-stream content type, got %q", chatRec.Header().Get("Content-Type"))
	}
	body := chatRec.Body.String()
	if !strings.Contains(body, "chat.completion.chunk") || !strings.Contains(body, "hello from gemini") {
		t.Fatalf("expected synthesized stream chunks, got %s", body)
	}
	if !strings.Contains(body, "data: [DONE]") {
		t.Fatalf("expected done marker, got %s", body)
	}

	logs, err := app.logStore.ListRecent(context.Background(), 10)
	if err != nil {
		t.Fatalf("list logs: %v", err)
	}
	if len(logs) != 1 || logs[0].TotalTokens != 5 {
		t.Fatalf("unexpected logs: %#v", logs)
	}
}

func seedKeyWithModel(t *testing.T, app *App, key types.UpstreamKey, model string) {
	t.Helper()

	createdKey, err := app.keyStore.Create(context.Background(), key)
	if err != nil {
		t.Fatalf("create key: %v", err)
	}
	if err := app.stateStore.Ensure(context.Background(), createdKey.ID); err != nil {
		t.Fatalf("ensure state: %v", err)
	}
	if err := app.modelStore.ReplaceForKey(context.Background(), createdKey.ID, []string{model}); err != nil {
		t.Fatalf("replace key models: %v", err)
	}
	if err := app.catalog.Rebuild(context.Background()); err != nil {
		t.Fatalf("rebuild catalog: %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }
