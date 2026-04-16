package probe

import (
	"context"
	"errors"
	"fmt"

	"github.com/zc12120/atomhub/internal/types"
)

var ErrUnsupportedProvider = errors.New("unsupported provider")

type keyStore interface {
	Get(context.Context, int64) (types.UpstreamKey, error)
	ListEnabled(context.Context) ([]types.UpstreamKey, error)
}

type modelStore interface {
	ReplaceForKey(context.Context, int64, []string) error
}

type stateStore interface {
	MarkProbeSuccess(context.Context, int64) error
	MarkProbeFailure(context.Context, int64, error) error
}

type catalogStore interface {
	Rebuild(context.Context) error
}

type Provider interface {
	ListModels(context.Context, types.UpstreamKey) ([]string, error)
}

type Service struct {
	keys    keyStore
	models  modelStore
	state   stateStore
	catalog catalogStore

	openai    Provider
	anthropic Provider
	gemini    Provider
}

func NewService(
	keys keyStore,
	models modelStore,
	state stateStore,
	catalog catalogStore,
	openai Provider,
	anthropic Provider,
	gemini Provider,
) *Service {
	return &Service{
		keys:      keys,
		models:    models,
		state:     state,
		catalog:   catalog,
		openai:    openai,
		anthropic: anthropic,
		gemini:    gemini,
	}
}

func (s *Service) ProbeKeyByID(ctx context.Context, keyID int64) error {
	key, err := s.keys.Get(ctx, keyID)
	if err != nil {
		return err
	}
	return s.ProbeKey(ctx, key)
}

func (s *Service) ProbeKey(ctx context.Context, key types.UpstreamKey) error {
	provider, err := s.providerFor(key.Provider)
	if err != nil {
		probeErr := fmt.Errorf("%w: %s", ErrUnsupportedProvider, key.Provider)
		_ = s.state.MarkProbeFailure(ctx, key.ID, probeErr)
		return probeErr
	}

	models, err := provider.ListModels(ctx, key)
	if err != nil {
		_ = s.state.MarkProbeFailure(ctx, key.ID, err)
		return err
	}
	if err := s.models.ReplaceForKey(ctx, key.ID, models); err != nil {
		return err
	}
	if err := s.state.MarkProbeSuccess(ctx, key.ID); err != nil {
		return err
	}
	return s.catalog.Rebuild(ctx)
}

func (s *Service) ProbeAll(ctx context.Context) map[int64]error {
	results := make(map[int64]error)
	keys, err := s.keys.ListEnabled(ctx)
	if err != nil {
		results[0] = err
		return results
	}
	for _, key := range keys {
		results[key.ID] = s.ProbeKey(ctx, key)
	}
	return results
}

func (s *Service) providerFor(provider types.Provider) (Provider, error) {
	switch provider {
	case types.ProviderOpenAI:
		if s.openai == nil {
			return nil, ErrUnsupportedProvider
		}
		return s.openai, nil
	case types.ProviderAnthropic:
		if s.anthropic == nil {
			return nil, ErrUnsupportedProvider
		}
		return s.anthropic, nil
	case types.ProviderGemini:
		if s.gemini == nil {
			return nil, ErrUnsupportedProvider
		}
		return s.gemini, nil
	default:
		return nil, ErrUnsupportedProvider
	}
}
