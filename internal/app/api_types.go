package app

import "time"

type openAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatRequest struct {
	Model       string              `json:"model"`
	Messages    []openAIChatMessage `json:"messages"`
	Temperature *float64            `json:"temperature,omitempty"`
	MaxTokens   *int                `json:"max_tokens,omitempty"`
	Stream      bool                `json:"stream,omitempty"`
}

type openAIChatChoice struct {
	Index        int               `json:"index"`
	Message      openAIChatMessage `json:"message"`
	FinishReason string            `json:"finish_reason"`
}

type openAIChatChunkDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

type openAIChatChunkChoice struct {
	Index        int                  `json:"index"`
	Delta        openAIChatChunkDelta `json:"delta"`
	FinishReason *string              `json:"finish_reason,omitempty"`
}

type openAIChatChunk struct {
	ID      string                  `json:"id"`
	Object  string                  `json:"object"`
	Created int64                   `json:"created"`
	Model   string                  `json:"model"`
	Choices []openAIChatChunkChoice `json:"choices"`
	Usage   map[string]int64        `json:"usage,omitempty"`
}

type openAIChatResponse struct {
	ID      string             `json:"id"`
	Object  string             `json:"object"`
	Created int64              `json:"created"`
	Model   string             `json:"model"`
	Choices []openAIChatChoice `json:"choices"`
	Usage   map[string]int64   `json:"usage,omitempty"`
}

type adminSessionResponse struct {
	Authenticated bool   `json:"authenticated"`
	Username      string `json:"username,omitempty"`
}

type adminKeyPayload struct {
	Name     string `json:"name"`
	Provider string `json:"provider"`
	BaseURL  string `json:"base_url"`
	APIKey   string `json:"api_key"`
	Enabled  *bool  `json:"enabled,omitempty"`
}

type adminKeyUpdatePayload struct {
	Name     *string `json:"name,omitempty"`
	Provider *string `json:"provider,omitempty"`
	BaseURL  *string `json:"base_url,omitempty"`
	APIKey   *string `json:"api_key,omitempty"`
	Enabled  *bool   `json:"enabled,omitempty"`
}

type adminKeyItem struct {
	ID         int64      `json:"id"`
	Provider   string     `json:"provider"`
	Label      string     `json:"label"`
	Status     string     `json:"status"`
	BaseURL    string     `json:"base_url,omitempty"`
	Enabled    bool       `json:"enabled"`
	LastError  string     `json:"last_error,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
}

type adminKeysResponse struct {
	Items []adminKeyItem `json:"items"`
}

type adminModelItem struct {
	Model       string `json:"model"`
	Provider    string `json:"provider"`
	KeyCount    int    `json:"key_count"`
	HealthyKeys int    `json:"healthy_keys"`
}

type adminModelsResponse struct {
	Items []adminModelItem `json:"items"`
}

type adminHealthSummary struct {
	HealthyKeys   int `json:"healthy_keys"`
	UnhealthyKeys int `json:"unhealthy_keys"`
	TotalKeys     int `json:"total_keys"`
}

type adminHealthResponse struct {
	Summary adminHealthSummary `json:"summary"`
	Keys    []adminKeyItem     `json:"keys"`
}

type adminDashboardResponse struct {
	Items   any                `json:"items"`
	Summary any                `json:"summary"`
	Health  adminHealthSummary `json:"health"`
}
