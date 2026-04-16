package gemini

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"

	"github.com/zc12120/atomhub/internal/providers/common"
	"github.com/zc12120/atomhub/internal/types"
)

const defaultBaseURL = "https://generativelanguage.googleapis.com"

type Provider struct {
	client *common.Client
}

func New(client *common.Client) *Provider {
	return &Provider{client: client}
}

func (p *Provider) ListModels(ctx context.Context, key types.UpstreamKey) ([]string, error) {
	baseURL := strings.TrimSpace(key.BaseURL)
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	endpoint, err := common.JoinURL(baseURL, "/v1beta/models")
	if err != nil {
		return nil, err
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	query := parsed.Query()
	query.Set("key", key.APIKey)
	parsed.RawQuery = query.Encode()

	var payload struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := p.client.GetJSON(ctx, parsed.String(), nil, &payload); err != nil {
		return nil, err
	}
	models := make([]string, 0, len(payload.Models))
	for _, model := range payload.Models {
		models = append(models, normalizeModelName(model.Name))
	}
	return uniqueOrdered(models), nil
}

func ParseModels(body []byte) ([]string, error) {
	var payload struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	models := make([]string, 0, len(payload.Models))
	for _, model := range payload.Models {
		models = append(models, normalizeModelName(model.Name))
	}
	return uniqueOrdered(models), nil
}

func normalizeModelName(name string) string {
	trimmed := strings.TrimSpace(name)
	if strings.HasPrefix(trimmed, "models/") {
		return strings.TrimPrefix(trimmed, "models/")
	}
	return trimmed
}

func uniqueOrdered(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, raw := range values {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
