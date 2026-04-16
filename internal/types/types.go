package types

import "time"

type Provider string

const (
	ProviderOpenAI    Provider = "openai"
	ProviderAnthropic Provider = "anthropic"
	ProviderGemini    Provider = "gemini"
)

type KeyStatus string

const (
	KeyStatusHealthy     KeyStatus = "healthy"
	KeyStatusDegraded    KeyStatus = "degraded"
	KeyStatusCoolingDown KeyStatus = "cooling_down"
	KeyStatusDisabled    KeyStatus = "disabled"
)

type UpstreamKey struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Provider  Provider  `json:"provider"`
	BaseURL   string    `json:"base_url"`
	APIKey    string    `json:"api_key,omitempty"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type KeyModel struct {
	ID        int64     `json:"id"`
	KeyID     int64     `json:"key_id"`
	Model     string    `json:"model"`
	CreatedAt time.Time `json:"created_at"`
}

type KeyState struct {
	KeyID               int64      `json:"key_id"`
	Status              KeyStatus  `json:"status"`
	CooldownUntil       *time.Time `json:"cooldown_until,omitempty"`
	ConsecutiveFailures int        `json:"consecutive_failures"`
	LastError           string     `json:"last_error,omitempty"`
	LastSuccessAt       *time.Time `json:"last_success_at,omitempty"`
	LastProbeAt         *time.Time `json:"last_probe_at,omitempty"`
}

type DownstreamKey struct {
	ID               int64      `json:"id"`
	Name             string     `json:"name"`
	TokenPrefix      string     `json:"token_prefix"`
	TokenHash        string     `json:"token_hash,omitempty"`
	EncryptedToken   string     `json:"-"`
	Enabled          bool       `json:"enabled"`
	LastUsedAt       *time.Time `json:"last_used_at,omitempty"`
	RequestCount     int64      `json:"request_count"`
	PromptTokens     int64      `json:"prompt_tokens"`
	CompletionTokens int64      `json:"completion_tokens"`
	TotalTokens      int64      `json:"total_tokens"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type KeyCandidate struct {
	KeyID         int64      `json:"key_id"`
	Status        KeyStatus  `json:"status"`
	CoolingDown   bool       `json:"cooling_down"`
	CooldownUntil *time.Time `json:"cooldown_until,omitempty"`
	Inflight      int        `json:"inflight"`
}

type RequestLog struct {
	ID               int64     `json:"id"`
	KeyID            int64     `json:"key_id"`
	DownstreamKeyID  *int64    `json:"downstream_key_id,omitempty"`
	Model            string    `json:"model"`
	PromptTokens     int64     `json:"prompt_tokens"`
	CompletionTokens int64     `json:"completion_tokens"`
	TotalTokens      int64     `json:"total_tokens"`
	LatencyMS        int64     `json:"latency_ms"`
	Status           string    `json:"status"`
	ErrorMessage     string    `json:"error_message,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}

type UsageTokens struct {
	PromptTokens     int64 `json:"prompt_tokens"`
	CompletionTokens int64 `json:"completion_tokens"`
	TotalTokens      int64 `json:"total_tokens"`
}

type ModelTokenStat struct {
	Model            string `json:"model"`
	PromptTokens     int64  `json:"prompt_tokens"`
	CompletionTokens int64  `json:"completion_tokens"`
	TotalTokens      int64  `json:"total_tokens"`
	RequestCount     int64  `json:"request_count"`
}

type TokenSummary struct {
	PromptTokens     int64 `json:"prompt_tokens"`
	CompletionTokens int64 `json:"completion_tokens"`
	TotalTokens      int64 `json:"total_tokens"`
	RequestCount     int64 `json:"request_count"`
}

type HealthOverview struct {
	Total       int `json:"total"`
	Healthy     int `json:"healthy"`
	Degraded    int `json:"degraded"`
	CoolingDown int `json:"cooling_down"`
	Disabled    int `json:"disabled"`
}
