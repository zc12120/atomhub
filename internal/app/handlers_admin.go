package app

import (
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/zc12120/atomhub/internal/auth"
	"github.com/zc12120/atomhub/internal/store"
	"github.com/zc12120/atomhub/internal/types"
)

func (a *App) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *App) handleAdminSession(w http.ResponseWriter, r *http.Request) {
	username, ok := a.sessionManager.Get(r)
	if !ok {
		writeJSON(w, http.StatusOK, adminSessionResponse{Authenticated: false})
		return
	}
	writeJSON(w, http.StatusOK, adminSessionResponse{Authenticated: true, Username: username})
}

func (a *App) handleAdminLogin(w http.ResponseWriter, r *http.Request) {
	var req adminLoginRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	user, err := a.adminRepo.Authenticate(r.Context(), req.Username, req.Password)
	if err != nil {
		if errors.Is(err, store.ErrInvalidCredentials) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "authentication failed"})
		return
	}

	if err := a.sessionManager.Set(w, user.Username); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create session"})
		return
	}

	writeJSON(w, http.StatusOK, adminSessionResponse{Authenticated: true, Username: user.Username})
}

func (a *App) handleAdminLogout(w http.ResponseWriter, _ *http.Request) {
	a.sessionManager.Clear(w)
	w.WriteHeader(http.StatusNoContent)
}

func (a *App) handleAdminMe(w http.ResponseWriter, r *http.Request) {
	username, ok := auth.UsernameFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"username": username})
}

func (a *App) handleDashboard(w http.ResponseWriter, r *http.Request) {
	items, summary, err := a.statsStore.TokenStats(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	overview, err := a.stateStore.Overview(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, adminDashboardResponse{
		Items:   items,
		Summary: summary,
		Health:  adminHealthSummary{HealthyKeys: overview.Healthy, UnhealthyKeys: overview.Degraded + overview.CoolingDown + overview.Disabled, TotalKeys: overview.Total},
	})
}

func (a *App) handleListKeys(w http.ResponseWriter, r *http.Request) {
	items, err := a.listAdminKeys(r)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, adminKeysResponse{Items: items})
}

func (a *App) handleCreateKey(w http.ResponseWriter, r *http.Request) {
	var payload adminKeyPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid payload"})
		return
	}
	key, err := payload.toUpstreamKey()
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	created, err := a.keyStore.Create(r.Context(), key)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if err := a.stateStore.Ensure(r.Context(), created.ID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	_ = a.probeService.ProbeKeyByID(r.Context(), created.ID)
	writeJSON(w, http.StatusCreated, mapAdminKey(created, types.KeyState{KeyID: created.ID, Status: types.KeyStatusHealthy}, nil))
}

func (a *App) handleUpdateKey(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDPath(w, r, "id")
	if !ok {
		return
	}
	var payload adminKeyUpdatePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid payload"})
		return
	}
	existing, err := a.keyStore.Get(r.Context(), id)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, store.ErrKeyNotFound) {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}
	key, err := payload.mergeInto(existing)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	updated, err := a.keyStore.Update(r.Context(), key)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, store.ErrKeyNotFound) {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}
	if err := a.stateStore.Ensure(r.Context(), updated.ID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if updated.Enabled {
		_ = a.probeService.ProbeKeyByID(r.Context(), updated.ID)
	} else {
		_ = a.catalog.Rebuild(r.Context())
	}
	state, _ := a.stateStore.Get(r.Context(), updated.ID)
	writeJSON(w, http.StatusOK, mapAdminKey(updated, state, nil))
}

func (a *App) handleDeleteKey(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDPath(w, r, "id")
	if !ok {
		return
	}
	if err := a.keyStore.Delete(r.Context(), id); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, store.ErrKeyNotFound) {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}
	_ = a.catalog.Rebuild(r.Context())
	w.WriteHeader(http.StatusNoContent)
}

func (a *App) handleProbeKey(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDPath(w, r, "id")
	if !ok {
		return
	}
	if err := a.probeService.ProbeKeyByID(r.Context(), id); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	key, err := a.keyStore.Get(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	state, _ := a.stateStore.Get(r.Context(), id)
	writeJSON(w, http.StatusOK, mapAdminKey(key, state, nil))
}

func (a *App) handleModels(w http.ResponseWriter, r *http.Request) {
	keys, err := a.keyStore.List(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	models, err := a.modelStore.ListAll(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	keyByID := make(map[int64]types.UpstreamKey, len(keys))
	for _, key := range keys {
		keyByID[key.ID] = key
	}
	type agg struct {
		item adminModelItem
		seen map[int64]struct{}
	}
	aggByPair := map[string]*agg{}
	for _, model := range models {
		key, ok := keyByID[model.KeyID]
		if !ok || !key.Enabled {
			continue
		}
		state, _ := a.stateStore.Get(r.Context(), key.ID)
		pair := string(key.Provider) + ":" + model.Model
		entry, ok := aggByPair[pair]
		if !ok {
			entry = &agg{item: adminModelItem{Model: model.Model, Provider: string(key.Provider)}, seen: map[int64]struct{}{}}
			aggByPair[pair] = entry
		}
		if _, seen := entry.seen[key.ID]; seen {
			continue
		}
		entry.seen[key.ID] = struct{}{}
		entry.item.KeyCount++
		if state.Status == types.KeyStatusHealthy || state.Status == "" {
			if state.CooldownUntil == nil || state.CooldownUntil.Before(time.Now().UTC()) {
				entry.item.HealthyKeys++
			}
		}
	}
	items := make([]adminModelItem, 0, len(aggByPair))
	for _, entry := range aggByPair {
		items = append(items, entry.item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Model == items[j].Model {
			return items[i].Provider < items[j].Provider
		}
		return items[i].Model < items[j].Model
	})
	writeJSON(w, http.StatusOK, adminModelsResponse{Items: items})
}

func (a *App) handleHealth(w http.ResponseWriter, r *http.Request) {
	keys, err := a.listAdminKeys(r)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	overview, err := a.stateStore.Overview(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, adminHealthResponse{
		Summary: adminHealthSummary{HealthyKeys: overview.Healthy, UnhealthyKeys: overview.Degraded + overview.CoolingDown + overview.Disabled, TotalKeys: overview.Total},
		Keys:    keys,
	})
}

func (a *App) handleRequests(w http.ResponseWriter, r *http.Request) {
	modelFilter := strings.TrimSpace(r.URL.Query().Get("model"))
	limit := 100
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		if parsed, err := strconv.Atoi(rawLimit); err == nil && parsed > 0 {
			if parsed > 500 {
				parsed = 500
			}
			limit = parsed
		}
	}

	logs, err := a.logStore.ListRecent(r.Context(), limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	keys, err := a.keyStore.List(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	keyByID := make(map[int64]types.UpstreamKey, len(keys))
	modelSet := make(map[string]struct{})
	for _, key := range keys {
		keyByID[key.ID] = key
	}
	items := make([]adminRequestLogItem, 0, len(logs))
	var summary adminRequestsSummary
	for _, entry := range logs {
		modelSet[entry.Model] = struct{}{}
		if modelFilter != "" && entry.Model != modelFilter {
			continue
		}
		key := keyByID[entry.KeyID]
		items = append(items, adminRequestLogItem{
			ID:               entry.ID,
			KeyID:            entry.KeyID,
			KeyLabel:         key.Name,
			Provider:         string(key.Provider),
			Model:            entry.Model,
			PromptTokens:     entry.PromptTokens,
			CompletionTokens: entry.CompletionTokens,
			TotalTokens:      entry.TotalTokens,
			LatencyMS:        entry.LatencyMS,
			Status:           entry.Status,
			ErrorMessage:     entry.ErrorMessage,
			CreatedAt:        entry.CreatedAt,
		})
		summary.RequestCount++
		summary.PromptTokens += entry.PromptTokens
		summary.CompletionTokens += entry.CompletionTokens
		summary.TotalTokens += entry.TotalTokens
		if entry.Status != "ok" {
			summary.ErrorCount++
		}
	}

	models := make([]string, 0, len(modelSet))
	for model := range modelSet {
		models = append(models, model)
	}
	sort.Strings(models)

	writeJSON(w, http.StatusOK, adminRequestsResponse{
		Items:   items,
		Summary: summary,
		Filters: adminRequestsFilters{Model: modelFilter, Models: models},
	})
}

func (a *App) listAdminKeys(r *http.Request) ([]adminKeyItem, error) {
	keys, err := a.keyStore.List(r.Context())
	if err != nil {
		return nil, err
	}
	recent, err := a.logStore.ListRecent(r.Context(), 500)
	if err != nil {
		return nil, err
	}
	lastUsed := make(map[int64]*time.Time)
	for _, row := range recent {
		if _, ok := lastUsed[row.KeyID]; ok {
			continue
		}
		used := row.CreatedAt
		lastUsed[row.KeyID] = &used
	}
	items := make([]adminKeyItem, 0, len(keys))
	for _, key := range keys {
		state, _ := a.stateStore.Get(r.Context(), key.ID)
		items = append(items, mapAdminKey(key, state, lastUsed[key.ID]))
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items, nil
}

func mapAdminKey(key types.UpstreamKey, state types.KeyState, lastUsedAt *time.Time) adminKeyItem {
	status := string(state.Status)
	if !key.Enabled {
		status = string(types.KeyStatusDisabled)
	}
	if status == "" {
		status = string(types.KeyStatusHealthy)
	}
	return adminKeyItem{
		ID:         key.ID,
		Provider:   string(key.Provider),
		Label:      key.Name,
		Status:     status,
		BaseURL:    key.BaseURL,
		Enabled:    key.Enabled,
		LastError:  state.LastError,
		LastUsedAt: lastUsedAt,
	}
}

func parseIDPath(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	raw := strings.TrimSpace(r.PathValue(name))
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return 0, false
	}
	return id, true
}

func (p adminKeyPayload) toUpstreamKey() (types.UpstreamKey, error) {
	provider, err := normalizeProvider(p.Provider)
	if err != nil {
		return types.UpstreamKey{}, err
	}
	enabled := true
	if p.Enabled != nil {
		enabled = *p.Enabled
	}
	name := strings.TrimSpace(p.Name)
	if name == "" {
		return types.UpstreamKey{}, errors.New("name is required")
	}
	if strings.TrimSpace(p.APIKey) == "" {
		return types.UpstreamKey{}, errors.New("api_key is required")
	}
	baseURL := strings.TrimSpace(p.BaseURL)
	if baseURL == "" {
		baseURL = defaultBaseURLForProvider(provider)
	}
	return types.UpstreamKey{Name: name, Provider: provider, BaseURL: baseURL, APIKey: strings.TrimSpace(p.APIKey), Enabled: enabled}, nil
}

func (p adminKeyUpdatePayload) mergeInto(existing types.UpstreamKey) (types.UpstreamKey, error) {
	updated := existing
	if p.Name != nil {
		name := strings.TrimSpace(*p.Name)
		if name == "" {
			return types.UpstreamKey{}, errors.New("name is required")
		}
		updated.Name = name
	}
	if p.Provider != nil {
		provider, err := normalizeProvider(*p.Provider)
		if err != nil {
			return types.UpstreamKey{}, err
		}
		updated.Provider = provider
	}
	if p.BaseURL != nil {
		updated.BaseURL = strings.TrimSpace(*p.BaseURL)
	}
	if p.APIKey != nil {
		apiKey := strings.TrimSpace(*p.APIKey)
		if apiKey == "" {
			return types.UpstreamKey{}, errors.New("api_key is required")
		}
		updated.APIKey = apiKey
	}
	if p.Enabled != nil {
		updated.Enabled = *p.Enabled
	}
	if strings.TrimSpace(updated.BaseURL) == "" {
		updated.BaseURL = defaultBaseURLForProvider(updated.Provider)
	}
	return updated, nil
}

func normalizeProvider(raw string) (types.Provider, error) {
	provider := types.Provider(strings.TrimSpace(strings.ToLower(raw)))
	switch provider {
	case types.ProviderOpenAI, types.ProviderAnthropic, types.ProviderGemini:
		return provider, nil
	default:
		return "", errors.New("provider must be openai, anthropic, or gemini")
	}
}

func defaultBaseURLForProvider(provider types.Provider) string {
	switch provider {
	case types.ProviderOpenAI:
		return "https://api.openai.com"
	case types.ProviderAnthropic:
		return "https://api.anthropic.com"
	case types.ProviderGemini:
		return "https://generativelanguage.googleapis.com"
	default:
		return ""
	}
}
