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

type adminRequestLogItem struct {
	ID                int64     `json:"id"`
	KeyID             int64     `json:"key_id"`
	KeyLabel          string    `json:"key_label,omitempty"`
	Provider          string    `json:"provider,omitempty"`
	DownstreamKeyID   *int64    `json:"downstream_key_id,omitempty"`
	DownstreamKeyName string    `json:"downstream_key_name,omitempty"`
	Model             string    `json:"model"`
	PromptTokens      int64     `json:"prompt_tokens"`
	CompletionTokens  int64     `json:"completion_tokens"`
	TotalTokens       int64     `json:"total_tokens"`
	LatencyMS         int64     `json:"latency_ms"`
	Status            string    `json:"status"`
	ErrorMessage      string    `json:"error_message,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}

type adminRequestsSummary struct {
	RequestCount     int64 `json:"request_count"`
	ErrorCount       int64 `json:"error_count"`
	PromptTokens     int64 `json:"prompt_tokens"`
	CompletionTokens int64 `json:"completion_tokens"`
	TotalTokens      int64 `json:"total_tokens"`
}

type adminRequestsFilters struct {
	Model  string   `json:"model,omitempty"`
	Models []string `json:"models"`
}

type adminRequestsResponse struct {
	Items   []adminRequestLogItem `json:"items"`
	Summary adminRequestsSummary  `json:"summary"`
	Filters adminRequestsFilters  `json:"filters"`
}

type adminDownstreamKeyPayload struct {
	Name    string `json:"name"`
	Enabled *bool  `json:"enabled,omitempty"`
}

type adminDownstreamKeyUpdatePayload struct {
	Name    *string `json:"name,omitempty"`
	Enabled *bool   `json:"enabled,omitempty"`
}

type adminDownstreamKeyItem struct {
	ID               int64      `json:"id"`
	Name             string     `json:"name"`
	TokenPrefix      string     `json:"token_prefix"`
	MaskedToken      string     `json:"masked_token"`
	CanReveal        bool       `json:"can_reveal"`
	Enabled          bool       `json:"enabled"`
	LastUsedAt       *time.Time `json:"last_used_at,omitempty"`
	RequestCount     int64      `json:"request_count"`
	PromptTokens     int64      `json:"prompt_tokens"`
	CompletionTokens int64      `json:"completion_tokens"`
	TotalTokens      int64      `json:"total_tokens"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type adminDownstreamKeysResponse struct {
	Items []adminDownstreamKeyItem `json:"items"`
}

type adminDownstreamKeyCreateResponse struct {
	Item  adminDownstreamKeyItem `json:"item"`
	Token string                 `json:"token"`
}

type adminDownstreamKeyTokenResponse struct {
	ID    int64  `json:"id"`
	Token string `json:"token"`
}
