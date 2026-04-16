package app

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/zc12120/atomhub/internal/types"
)

type sseResponseWriter struct {
	writer  http.ResponseWriter
	flusher http.Flusher
	started bool
}

func newSSEResponseWriter(w http.ResponseWriter) *sseResponseWriter {
	flusher, _ := w.(http.Flusher)
	return &sseResponseWriter{writer: w, flusher: flusher}
}

func (s *sseResponseWriter) Started() bool { return s.started }

func (s *sseResponseWriter) WriteData(payload string) error {
	if !s.started {
		headers := s.writer.Header()
		headers.Set("Content-Type", "text/event-stream")
		headers.Set("Cache-Control", "no-cache")
		headers.Set("Connection", "keep-alive")
		headers.Set("X-Accel-Buffering", "no")
		s.writer.WriteHeader(http.StatusOK)
		s.started = true
	}
	if _, err := io.WriteString(s.writer, "data: "+payload+"\n\n"); err != nil {
		return err
	}
	if s.flusher != nil {
		s.flusher.Flush()
	}
	return nil
}

func (s *sseResponseWriter) WriteJSONData(payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return s.WriteData(string(body))
}

func (a *App) streamChatCompletion(w http.ResponseWriter, r *http.Request, key types.UpstreamKey, req openAIChatRequest) (types.UsageTokens, bool, error) {
	stream := newSSEResponseWriter(w)
	var (
		usageTokens types.UsageTokens
		err         error
	)
	switch key.Provider {
	case types.ProviderOpenAI:
		usageTokens, err = a.streamOpenAI(r, key, req, stream)
	case types.ProviderAnthropic, types.ProviderGemini:
		usageTokens, err = a.streamFromSingleResponse(r, key, req, stream)
	default:
		err = fmt.Errorf("unsupported provider: %s", key.Provider)
	}
	return usageTokens, stream.Started(), err
}

func (a *App) streamOpenAI(r *http.Request, key types.UpstreamKey, req openAIChatRequest, stream *sseResponseWriter) (types.UsageTokens, error) {
	endpoint, err := joinURL(defaultString(key.BaseURL, "https://api.openai.com"), "/v1/chat/completions")
	if err != nil {
		return types.UsageTokens{}, err
	}
	req.Stream = true
	body, _ := json.Marshal(req)
	httpReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return types.UsageTokens{}, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+key.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := a.upstreamClient.Do(httpReq)
	if err != nil {
		return types.UsageTokens{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		responseBody, _ := io.ReadAll(resp.Body)
		return types.UsageTokens{}, fmt.Errorf("openai upstream returned %d: %s", resp.StatusCode, string(responseBody))
	}

	var usageTokens types.UsageTokens
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" {
			continue
		}
		if payload == "[DONE]" {
			if err := stream.WriteData("[DONE]"); err != nil {
				return usageTokens, err
			}
			return usageTokens, nil
		}

		var chunk openAIChatChunk
		if err := json.Unmarshal([]byte(payload), &chunk); err == nil {
			usageTokens = mergeUsageTokens(usageTokens, chunk.Usage)
		}
		if err := stream.WriteData(payload); err != nil {
			return usageTokens, err
		}
	}
	if err := scanner.Err(); err != nil {
		return usageTokens, err
	}
	if err := stream.WriteData("[DONE]"); err != nil {
		return usageTokens, err
	}
	return usageTokens, nil
}

func (a *App) streamFromSingleResponse(r *http.Request, key types.UpstreamKey, req openAIChatRequest, stream *sseResponseWriter) (types.UsageTokens, error) {
	nonStreamingReq := req
	nonStreamingReq.Stream = false

	var (
		response    openAIChatResponse
		usageTokens types.UsageTokens
		err         error
	)
	switch key.Provider {
	case types.ProviderAnthropic:
		response, usageTokens, err = a.proxyAnthropic(r, key, nonStreamingReq)
	case types.ProviderGemini:
		response, usageTokens, err = a.proxyGemini(r, key, nonStreamingReq)
	default:
		return types.UsageTokens{}, fmt.Errorf("stream fallback unsupported for provider: %s", key.Provider)
	}
	if err != nil {
		return types.UsageTokens{}, err
	}

	chunkID := defaultString(response.ID, fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano()))
	model := defaultString(response.Model, req.Model)
	created := response.Created
	if created == 0 {
		created = time.Now().Unix()
	}

	message := openAIChatMessage{Role: "assistant"}
	finishReason := "stop"
	if len(response.Choices) > 0 {
		message = response.Choices[0].Message
		if strings.TrimSpace(response.Choices[0].FinishReason) != "" {
			finishReason = response.Choices[0].FinishReason
		}
	}

	initialChunk := openAIChatChunk{
		ID:      chunkID,
		Object:  "chat.completion.chunk",
		Created: created,
		Model:   model,
		Choices: []openAIChatChunkChoice{
			{
				Index: 0,
				Delta: openAIChatChunkDelta{
					Role:    defaultString(message.Role, "assistant"),
					Content: message.Content,
				},
			},
		},
	}
	if err := stream.WriteJSONData(initialChunk); err != nil {
		return usageTokens, err
	}

	finalChunk := openAIChatChunk{
		ID:      chunkID,
		Object:  "chat.completion.chunk",
		Created: created,
		Model:   model,
		Choices: []openAIChatChunkChoice{
			{
				Index:        0,
				Delta:        openAIChatChunkDelta{},
				FinishReason: &finishReason,
			},
		},
	}
	if err := stream.WriteJSONData(finalChunk); err != nil {
		return usageTokens, err
	}
	if err := stream.WriteData("[DONE]"); err != nil {
		return usageTokens, err
	}
	return usageTokens, nil
}

func mergeUsageTokens(current types.UsageTokens, incoming map[string]int64) types.UsageTokens {
	if incoming == nil {
		return current
	}
	if value := incoming["prompt_tokens"]; value > current.PromptTokens {
		current.PromptTokens = value
	}
	if value := incoming["completion_tokens"]; value > current.CompletionTokens {
		current.CompletionTokens = value
	}
	if value := incoming["total_tokens"]; value > current.TotalTokens {
		current.TotalTokens = value
	}
	return current
}
