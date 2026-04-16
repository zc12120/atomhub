package catalog

import (
	"context"
	"sort"
	"sync"

	"github.com/zc12120/atomhub/internal/types"
)

type keySource interface {
	List(context.Context) ([]types.UpstreamKey, error)
}

type modelSource interface {
	ListAll(context.Context) ([]types.KeyModel, error)
}

type Catalog struct {
	keys   keySource
	models modelSource

	mu          sync.RWMutex
	keysByModel map[string][]int64
}

func New(keys keySource, models modelSource) *Catalog {
	return &Catalog{
		keys:        keys,
		models:      models,
		keysByModel: make(map[string][]int64),
	}
}

func (c *Catalog) Rebuild(ctx context.Context) error {
	keys, err := c.keys.List(ctx)
	if err != nil {
		return err
	}
	models, err := c.models.ListAll(ctx)
	if err != nil {
		return err
	}

	enabled := make(map[int64]struct{}, len(keys))
	for _, key := range keys {
		if key.Enabled {
			enabled[key.ID] = struct{}{}
		}
	}

	next := make(map[string][]int64)
	for _, entry := range models {
		if _, ok := enabled[entry.KeyID]; !ok {
			continue
		}
		existing := next[entry.Model]
		if containsKeyID(existing, entry.KeyID) {
			continue
		}
		next[entry.Model] = append(existing, entry.KeyID)
	}
	for model := range next {
		sort.Slice(next[model], func(i, j int) bool {
			return next[model][i] < next[model][j]
		})
	}

	c.mu.Lock()
	c.keysByModel = next
	c.mu.Unlock()
	return nil
}

func (c *Catalog) KeysForModel(model string) []int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return append([]int64(nil), c.keysByModel[model]...)
}

func (c *Catalog) Snapshot() map[string][]int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make(map[string][]int64, len(c.keysByModel))
	for model, ids := range c.keysByModel {
		out[model] = append([]int64(nil), ids...)
	}
	return out
}

func containsKeyID(ids []int64, keyID int64) bool {
	for _, existing := range ids {
		if existing == keyID {
			return true
		}
	}
	return false
}
