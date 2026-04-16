package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	internalauth "github.com/zc12120/atomhub/internal/auth"
	"github.com/zc12120/atomhub/internal/selector"
	"github.com/zc12120/atomhub/internal/store"
	"github.com/zc12120/atomhub/internal/types"
	"github.com/zc12120/atomhub/internal/usage"
)

func (a *App) requireGatewayAccess(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := bearerToken(strings.TrimSpace(r.Header.Get("Authorization")))
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		if a.Config.GatewayToken != "" && token == a.Config.GatewayToken {
			next.ServeHTTP(w, r)
			return
		}
		downstreamKey, err := a.downstreamKeyStore.FindByToken(r.Context(), token)
		if err != nil {
			if err == store.ErrDownstreamKeyNotFound {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "authenticate gateway request failed"})
			return
		}
		next.ServeHTTP(w, r.WithContext(internalauth.WithDownstreamKey(r.Context(), downstreamKey)))
	})
}

func bearerToken(authz string) (string, bool) {
	if authz == "" {
		return "", false
	}
	parts := strings.SplitN(authz, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(strings.TrimSpace(parts[0]), "Bearer") {
		return "", false
	}
	token := strings.TrimSpace(parts[1])
	return token, token != ""
}

func (a *App) handleGatewayModels(w http.ResponseWriter, _ *http.Request) {
	snapshot := a.catalog.Snapshot()
	models := make([]string, 0, len(snapshot))
	for model := range snapshot {
		models = append(models, model)
	}
	sort.Strings(models)
	data := make([]map[string]any, 0, len(models))
	for _, model := range models {
		data = append(data, map[string]any{
			"id":       model,
			"object":   "model",
			"owned_by": "atomhub",
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"object": "list", "data": data})
}

func (a *App) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	var req openAIChatRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if strings.TrimSpace(req.Model) == "" || len(req.Messages) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "model and messages are required"})
		return
	}

	keyIDs := a.catalog.KeysForModel(req.Model)
	if len(keyIDs) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "no key supports requested model"})
		return
	}
	candidateStates, err := a.stateStore.Candidates(r.Context(), keyIDs)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	candidates := make([]selector.Candidate, 0, len(candidateStates))
	for _, candidate := range candidateStates {
		candidates = append(candidates, selector.Candidate{KeyID: candidate.KeyID, CoolingDown: candidate.CoolingDown, Inflight: candidate.Inflight})
	}
	selected, err := a.selector.Select(candidates)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	key, err := a.keyStore.Get(r.Context(), selected.KeyID)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}

	a.stateStore.IncrementInflight(selected.KeyID)
	defer a.stateStore.DecrementInflight(selected.KeyID)

	if req.Stream {
		started := time.Now()
		streamUsage, responseStarted, streamErr := a.streamChatCompletion(w, r, key, req)
		latency := time.Since(started)
		normalized := usage.Normalize(usage.ParsedUsage{PromptTokens: streamUsage.PromptTokens, CompletionTokens: streamUsage.CompletionTokens, TotalTokens: streamUsage.TotalTokens})
		usageTokens := types.UsageTokens{PromptTokens: normalized.PromptTokens, CompletionTokens: normalized.CompletionTokens, TotalTokens: normalized.TotalTokens}
		_, _ = a.logStore.Insert(r.Context(), selected.KeyID, downstreamKeyIDFromContext(r.Context()), req.Model, usageTokens, latency, streamErr)
		if streamErr != nil {
			_ = a.stateStore.MarkFailure(r.Context(), selected.KeyID, streamErr)
			if !responseStarted {
				writeJSON(w, http.StatusBadGateway, map[string]string{"error": streamErr.Error()})
			}
			return
		}
		_ = a.stateStore.MarkSuccess(r.Context(), selected.KeyID)
		return
	}

	started := time.Now()
	result, tokenUsage, upstreamErr := a.proxyChatCompletion(r, key, req)
	latency := time.Since(started)
	normalized := usage.Normalize(usage.ParsedUsage{PromptTokens: tokenUsage.PromptTokens, CompletionTokens: tokenUsage.CompletionTokens, TotalTokens: tokenUsage.TotalTokens})
	usageTokens := types.UsageTokens{PromptTokens: normalized.PromptTokens, CompletionTokens: normalized.CompletionTokens, TotalTokens: normalized.TotalTokens}
	_, _ = a.logStore.Insert(r.Context(), selected.KeyID, downstreamKeyIDFromContext(r.Context()), req.Model, usageTokens, latency, upstreamErr)
	if upstreamErr != nil {
		_ = a.stateStore.MarkFailure(r.Context(), selected.KeyID, upstreamErr)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": upstreamErr.Error()})
		return
	}
	_ = a.stateStore.MarkSuccess(r.Context(), selected.KeyID)
	writeJSON(w, http.StatusOK, result)
}

func downstreamKeyIDFromContext(ctx context.Context) *int64 {
	downstreamKey, ok := internalauth.DownstreamKeyFromContext(ctx)
	if !ok || downstreamKey.ID == 0 {
		return nil
	}
	id := downstreamKey.ID
	return &id
}

func (a *App) proxyChatCompletion(r *http.Request, key types.UpstreamKey, req openAIChatRequest) (openAIChatResponse, types.UsageTokens, error) {
	switch key.Provider {
	case types.ProviderOpenAI:
		return a.proxyOpenAI(r, key, req)
	case types.ProviderAnthropic:
		return a.proxyAnthropic(r, key, req)
	case types.ProviderGemini:
		return a.proxyGemini(r, key, req)
	default:
		return openAIChatResponse{}, types.UsageTokens{}, fmt.Errorf("unsupported provider: %s", key.Provider)
	}
}

func (a *App) proxyOpenAI(r *http.Request, key types.UpstreamKey, req openAIChatRequest) (openAIChatResponse, types.UsageTokens, error) {
	endpoint, err := joinURL(defaultString(key.BaseURL, "https://api.openai.com"), "/v1/chat/completions")
	if err != nil {
		return openAIChatResponse{}, types.UsageTokens{}, err
	}
	body, _ := json.Marshal(req)
	httpReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return openAIChatResponse{}, types.UsageTokens{}, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+key.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := a.upstreamClient.Do(httpReq)
	if err != nil {
		return openAIChatResponse{}, types.UsageTokens{}, err
	}
	defer resp.Body.Close()
	responseBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return openAIChatResponse{}, types.UsageTokens{}, fmt.Errorf("openai upstream returned %d: %s", resp.StatusCode, string(responseBody))
	}
	var parsed openAIChatResponse
	if err := json.Unmarshal(responseBody, &parsed); err != nil {
		return openAIChatResponse{}, types.UsageTokens{}, err
	}
	return parsed, types.UsageTokens{PromptTokens: parsed.Usage["prompt_tokens"], CompletionTokens: parsed.Usage["completion_tokens"], TotalTokens: parsed.Usage["total_tokens"]}, nil
}

func (a *App) proxyAnthropic(r *http.Request, key types.UpstreamKey, req openAIChatRequest) (openAIChatResponse, types.UsageTokens, error) {
	endpoint, err := joinURL(defaultString(key.BaseURL, "https://api.anthropic.com"), "/v1/messages")
	if err != nil {
		return openAIChatResponse{}, types.UsageTokens{}, err
	}
	system, messages := splitSystemMessages(req.Messages)
	payload := map[string]any{
		"model":      req.Model,
		"messages":   messages,
		"max_tokens": coalesceMaxTokens(req.MaxTokens, 1024),
	}
	if system != "" {
		payload["system"] = system
	}
	if req.Temperature != nil {
		payload["temperature"] = *req.Temperature
	}
	body, _ := json.Marshal(payload)
	httpReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return openAIChatResponse{}, types.UsageTokens{}, err
	}
	httpReq.Header.Set("x-api-key", key.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := a.upstreamClient.Do(httpReq)
	if err != nil {
		return openAIChatResponse{}, types.UsageTokens{}, err
	}
	defer resp.Body.Close()
	responseBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return openAIChatResponse{}, types.UsageTokens{}, fmt.Errorf("anthropic upstream returned %d: %s", resp.StatusCode, string(responseBody))
	}
	var parsed struct {
		ID    string `json:"id"`
		Model string `json:"model"`
		Usage struct {
			InputTokens  int64 `json:"input_tokens"`
			OutputTokens int64 `json:"output_tokens"`
		} `json:"usage"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		StopReason string `json:"stop_reason"`
	}
	if err := json.Unmarshal(responseBody, &parsed); err != nil {
		return openAIChatResponse{}, types.UsageTokens{}, err
	}
	parts := make([]string, 0, len(parsed.Content))
	for _, item := range parsed.Content {
		if item.Type == "text" && strings.TrimSpace(item.Text) != "" {
			parts = append(parts, item.Text)
		}
	}
	finish := "stop"
	if parsed.StopReason == "max_tokens" {
		finish = "length"
	}
	out := openAIChatResponse{ID: parsed.ID, Object: "chat.completion", Created: time.Now().Unix(), Model: defaultString(parsed.Model, req.Model), Choices: []openAIChatChoice{{Index: 0, Message: openAIChatMessage{Role: "assistant", Content: strings.Join(parts, "\n")}, FinishReason: finish}}, Usage: map[string]int64{"prompt_tokens": parsed.Usage.InputTokens, "completion_tokens": parsed.Usage.OutputTokens, "total_tokens": parsed.Usage.InputTokens + parsed.Usage.OutputTokens}}
	return out, types.UsageTokens{PromptTokens: parsed.Usage.InputTokens, CompletionTokens: parsed.Usage.OutputTokens, TotalTokens: parsed.Usage.InputTokens + parsed.Usage.OutputTokens}, nil
}

func (a *App) proxyGemini(r *http.Request, key types.UpstreamKey, req openAIChatRequest) (openAIChatResponse, types.UsageTokens, error) {
	base := defaultString(key.BaseURL, "https://generativelanguage.googleapis.com")
	endpoint, err := joinURL(base, "/v1beta/models/"+req.Model+":generateContent")
	if err != nil {
		return openAIChatResponse{}, types.UsageTokens{}, err
	}
	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		return openAIChatResponse{}, types.UsageTokens{}, err
	}
	query := parsedURL.Query()
	query.Set("key", key.APIKey)
	parsedURL.RawQuery = query.Encode()
	system, messages := splitSystemMessages(req.Messages)
	contents := make([]map[string]any, 0, len(messages))
	if system != "" {
		contents = append(contents, map[string]any{"role": "user", "parts": []map[string]string{{"text": system}}})
	}
	for _, msg := range messages {
		role := "user"
		if msg.Role == "assistant" {
			role = "model"
		}
		contents = append(contents, map[string]any{"role": role, "parts": []map[string]string{{"text": msg.Content}}})
	}
	payload := map[string]any{"contents": contents}
	generationConfig := map[string]any{}
	if req.Temperature != nil {
		generationConfig["temperature"] = *req.Temperature
	}
	if req.MaxTokens != nil {
		generationConfig["maxOutputTokens"] = *req.MaxTokens
	}
	if len(generationConfig) > 0 {
		payload["generationConfig"] = generationConfig
	}
	body, _ := json.Marshal(payload)
	httpReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost, parsedURL.String(), bytes.NewReader(body))
	if err != nil {
		return openAIChatResponse{}, types.UsageTokens{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := a.upstreamClient.Do(httpReq)
	if err != nil {
		return openAIChatResponse{}, types.UsageTokens{}, err
	}
	defer resp.Body.Close()
	responseBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return openAIChatResponse{}, types.UsageTokens{}, fmt.Errorf("gemini upstream returned %d: %s", resp.StatusCode, string(responseBody))
	}
	var parsed struct {
		Candidates []struct {
			FinishReason string `json:"finishReason"`
			Content      struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		UsageMetadata struct {
			PromptTokenCount     int64 `json:"promptTokenCount"`
			CandidatesTokenCount int64 `json:"candidatesTokenCount"`
			TotalTokenCount      int64 `json:"totalTokenCount"`
		} `json:"usageMetadata"`
	}
	if err := json.Unmarshal(responseBody, &parsed); err != nil {
		return openAIChatResponse{}, types.UsageTokens{}, err
	}
	text := ""
	finish := "stop"
	if len(parsed.Candidates) > 0 {
		parts := make([]string, 0, len(parsed.Candidates[0].Content.Parts))
		for _, part := range parsed.Candidates[0].Content.Parts {
			if strings.TrimSpace(part.Text) != "" {
				parts = append(parts, part.Text)
			}
		}
		text = strings.Join(parts, "\n")
		if strings.Contains(strings.ToLower(parsed.Candidates[0].FinishReason), "max") {
			finish = "length"
		}
	}
	usageMap := map[string]int64{"prompt_tokens": parsed.UsageMetadata.PromptTokenCount, "completion_tokens": parsed.UsageMetadata.CandidatesTokenCount, "total_tokens": parsed.UsageMetadata.TotalTokenCount}
	if usageMap["total_tokens"] == 0 {
		usageMap["total_tokens"] = usageMap["prompt_tokens"] + usageMap["completion_tokens"]
	}
	out := openAIChatResponse{ID: uuid.NewString(), Object: "chat.completion", Created: time.Now().Unix(), Model: req.Model, Choices: []openAIChatChoice{{Index: 0, Message: openAIChatMessage{Role: "assistant", Content: text}, FinishReason: finish}}, Usage: usageMap}
	return out, types.UsageTokens{PromptTokens: usageMap["prompt_tokens"], CompletionTokens: usageMap["completion_tokens"], TotalTokens: usageMap["total_tokens"]}, nil
}

func splitSystemMessages(messages []openAIChatMessage) (string, []openAIChatMessage) {
	systemParts := make([]string, 0)
	filtered := make([]openAIChatMessage, 0, len(messages))
	for _, msg := range messages {
		if strings.EqualFold(msg.Role, "system") {
			if strings.TrimSpace(msg.Content) != "" {
				systemParts = append(systemParts, msg.Content)
			}
			continue
		}
		filtered = append(filtered, msg)
	}
	return strings.Join(systemParts, "\n"), filtered
}

func coalesceMaxTokens(v *int, fallback int) int {
	if v == nil || *v <= 0 {
		return fallback
	}
	return *v
}

func defaultString(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return strings.TrimSpace(v)
}

func joinURL(baseURL, suffix string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return "", err
	}
	parsed.Path = path.Join(parsed.Path, suffix)
	return parsed.String(), nil
}
