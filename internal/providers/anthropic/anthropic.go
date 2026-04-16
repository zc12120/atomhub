package anthropic

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/zc12120/atomhub/internal/providers/common"
	"github.com/zc12120/atomhub/internal/types"
)

const defaultBaseURL = "https://api.anthropic.com"

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
	endpoint, err := common.JoinURL(baseURL, "/v1/models")
	if err != nil {
		return nil, err
	}
	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := p.client.GetJSON(ctx, endpoint, map[string]string{
		"x-api-key":         key.APIKey,
		"anthropic-version": "2023-06-01",
	}, &payload); err != nil {
		return nil, err
	}
	models := make([]string, 0, len(payload.Data))
	for _, model := range payload.Data {
		models = append(models, model.ID)
	}
	return uniqueOrdered(models), nil
}

func ParseModels(body []byte) ([]string, error) {
	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	models := make([]string, 0, len(payload.Data))
	for _, model := range payload.Data {
		models = append(models, model.ID)
	}
	return uniqueOrdered(models), nil
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
