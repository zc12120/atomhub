# AtomHub Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a self-hosted Go + SQLite + web UI gateway that manages OpenAI/Anthropic/Gemini upstream keys, probes models, load-balances same-model traffic behind an OpenAI-compatible downstream API, records token usage, and exposes an authenticated admin dashboard.

**Architecture:** A single Go binary serves both the OpenAI-compatible gateway API and the admin API/UI. SQLite persists keys, model support, key health, request logs, and admin auth data, while in-memory indexes accelerate model-to-key routing. A lightweight Vite/React frontend is built into static assets and served by the Go backend after login via an HTTP-only session cookie.

**Tech Stack:** Go 1.24+, net/http, database/sql + modernc.org/sqlite, golang.org/x/crypto/bcrypt, React + Vite + TypeScript, Vitest for frontend smoke tests, Go testing package, Docker multi-stage build.

---

## File Structure

### Backend bootstrap
- Create: `go.mod`
- Create: `cmd/atomhub/main.go`
- Create: `internal/app/app.go`
- Create: `internal/config/config.go`

### Persistence and domain
- Create: `internal/store/db.go`
- Create: `internal/store/migrate.go`
- Create: `internal/store/admin.go`
- Create: `internal/store/keys.go`
- Create: `internal/store/models.go`
- Create: `internal/store/state.go`
- Create: `internal/store/logs.go`
- Create: `internal/store/stats.go`
- Create: `internal/types/types.go`

### Auth and sessions
- Create: `internal/auth/password.go`
- Create: `internal/auth/session.go`
- Create: `internal/auth/middleware.go`
- Test: `internal/auth/session_test.go`

### Routing, probing, proxying
- Create: `internal/catalog/catalog.go`
- Create: `internal/selector/selector.go`
- Create: `internal/probe/service.go`
- Create: `internal/usage/usage.go`
- Create: `internal/providers/common/http.go`
- Create: `internal/providers/openai/openai.go`
- Create: `internal/providers/anthropic/anthropic.go`
- Create: `internal/providers/gemini/gemini.go`
- Create: `internal/httpapi/router.go`
- Create: `internal/httpapi/admin_auth.go`
- Create: `internal/httpapi/admin_keys.go`
- Create: `internal/httpapi/admin_dashboard.go`
- Create: `internal/httpapi/admin_health.go`
- Create: `internal/httpapi/openai_compat.go`
- Test: `internal/selector/selector_test.go`
- Test: `internal/usage/usage_test.go`
- Test: `internal/providers/anthropic/anthropic_test.go`
- Test: `internal/providers/gemini/gemini_test.go`

### Frontend
- Create: `web/package.json`
- Create: `web/tsconfig.json`
- Create: `web/vite.config.ts`
- Create: `web/index.html`
- Create: `web/src/main.tsx`
- Create: `web/src/App.tsx`
- Create: `web/src/api.ts`
- Create: `web/src/auth.ts`
- Create: `web/src/styles.css`
- Create: `web/src/pages/LoginPage.tsx`
- Create: `web/src/pages/DashboardPage.tsx`
- Create: `web/src/pages/KeysPage.tsx`
- Create: `web/src/pages/ModelsPage.tsx`
- Create: `web/src/pages/HealthPage.tsx`
- Create: `web/src/components/Layout.tsx`
- Create: `web/src/components/StatCard.tsx`
- Test: `web/src/pages/DashboardPage.test.tsx`

### Build and deploy
- Create: `Makefile`
- Create: `Dockerfile`
- Create: `docker-compose.yml`
- Create: `.env.example`
- Modify: `README.md`

---

### Task 1: Bootstrap backend skeleton and persistent admin auth

**Files:**
- Create: `go.mod`
- Create: `cmd/atomhub/main.go`
- Create: `internal/app/app.go`
- Create: `internal/config/config.go`
- Create: `internal/store/db.go`
- Create: `internal/store/migrate.go`
- Create: `internal/store/admin.go`
- Create: `internal/auth/password.go`
- Create: `internal/auth/session.go`
- Create: `internal/auth/middleware.go`
- Test: `internal/auth/session_test.go`

- [ ] **Step 1: Write the failing test**

```go
package auth

import (
    "net/http/httptest"
    "testing"
    "time"
)

func TestSessionCookieRoundTrip(t *testing.T) {
    manager := NewSessionManager("test-secret", 24*time.Hour)
    rr := httptest.NewRecorder()
    if err := manager.Set(rr, "admin"); err != nil {
        t.Fatalf("set session: %v", err)
    }
    req := httptest.NewRequest("GET", "/admin", nil)
    for _, cookie := range rr.Result().Cookies() {
        req.AddCookie(cookie)
    }
    username, ok := manager.Get(req)
    if !ok || username != "admin" {
        t.Fatalf("expected admin session, got %q ok=%v", username, ok)
    }
}

func TestPasswordHashRoundTrip(t *testing.T) {
    hash, err := HashPassword("changeme")
    if err != nil {
        t.Fatalf("hash password: %v", err)
    }
    if err := VerifyPassword(hash, "changeme"); err != nil {
        t.Fatalf("verify password: %v", err)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/auth -v`
Expected: FAIL with undefined symbols for `NewSessionManager`, `HashPassword`, and `VerifyPassword`.

- [ ] **Step 3: Write minimal implementation**

```go
// internal/auth/password.go
package auth

import "golang.org/x/crypto/bcrypt"

func HashPassword(password string) (string, error) {
    hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        return "", err
    }
    return string(hashed), nil
}

func VerifyPassword(hash string, password string) error {
    return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
```

```go
// internal/auth/session.go
package auth

import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/base64"
    "fmt"
    "net/http"
    "strconv"
    "strings"
    "time"
)

type SessionManager struct {
    secret []byte
    ttl    time.Duration
}

func NewSessionManager(secret string, ttl time.Duration) *SessionManager {
    return &SessionManager{secret: []byte(secret), ttl: ttl}
}

func (m *SessionManager) Set(w http.ResponseWriter, username string) error {
    expires := time.Now().Add(m.ttl).Unix()
    payload := fmt.Sprintf("%s|%d", username, expires)
    mac := hmac.New(sha256.New, m.secret)
    mac.Write([]byte(payload))
    sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
    token := base64.RawURLEncoding.EncodeToString([]byte(payload + "|" + sig))
    http.SetCookie(w, &http.Cookie{Name: "atomhub_session", Value: token, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode})
    return nil
}

func (m *SessionManager) Get(r *http.Request) (string, bool) {
    c, err := r.Cookie("atomhub_session")
    if err != nil {
        return "", false
    }
    raw, err := base64.RawURLEncoding.DecodeString(c.Value)
    if err != nil {
        return "", false
    }
    parts := strings.Split(string(raw), "|")
    if len(parts) != 3 {
        return "", false
    }
    payload := parts[0] + "|" + parts[1]
    mac := hmac.New(sha256.New, m.secret)
    mac.Write([]byte(payload))
    if base64.RawURLEncoding.EncodeToString(mac.Sum(nil)) != parts[2] {
        return "", false
    }
    expiry, err := strconv.ParseInt(parts[1], 10, 64)
    if err != nil || time.Now().Unix() > expiry {
        return "", false
    }
    return parts[0], true
}
```

- [ ] **Step 4: Add database bootstrap**

```go
// internal/store/migrate.go
package store

import "database/sql"

func Migrate(db *sql.DB) error {
    stmts := []string{
        `create table if not exists admin_users (
            id integer primary key autoincrement,
            username text not null unique,
            password_hash text not null,
            created_at text not null default current_timestamp
        );`,
        `create table if not exists upstream_keys (
            id integer primary key autoincrement,
            name text not null,
            provider text not null,
            base_url text not null,
            api_key text not null,
            enabled integer not null default 1,
            created_at text not null default current_timestamp,
            updated_at text not null default current_timestamp
        );`,
        `create table if not exists key_models (
            id integer primary key autoincrement,
            key_id integer not null,
            model text not null,
            created_at text not null default current_timestamp,
            unique(key_id, model)
        );`,
        `create table if not exists key_state (
            key_id integer primary key,
            status text not null default 'healthy',
            cooldown_until text,
            consecutive_failures integer not null default 0,
            last_error text,
            last_success_at text,
            last_probe_at text
        );`,
        `create table if not exists request_logs (
            id integer primary key autoincrement,
            key_id integer not null,
            model text not null,
            prompt_tokens integer not null default 0,
            completion_tokens integer not null default 0,
            total_tokens integer not null default 0,
            latency_ms integer not null default 0,
            status text not null,
            error_message text,
            created_at text not null default current_timestamp
        );`,
    }
    for _, stmt := range stmts {
        if _, err := db.Exec(stmt); err != nil {
            return err
        }
    }
    return nil
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/auth -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add go.mod cmd/atomhub/main.go internal/app internal/config internal/store internal/auth
git commit -m "feat: bootstrap backend auth and sqlite"
```

### Task 2: Implement key registry, provider probing, and in-memory catalog

**Files:**
- Create: `internal/types/types.go`
- Create: `internal/store/keys.go`
- Create: `internal/store/models.go`
- Create: `internal/store/state.go`
- Create: `internal/catalog/catalog.go`
- Create: `internal/probe/service.go`
- Create: `internal/providers/common/http.go`
- Create: `internal/providers/openai/openai.go`
- Create: `internal/providers/anthropic/anthropic.go`
- Create: `internal/providers/gemini/gemini.go`
- Test: `internal/providers/anthropic/anthropic_test.go`
- Test: `internal/providers/gemini/gemini_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestAnthropicModelsResponseParsesModels(t *testing.T) {
    body := []byte(`{"data":[{"id":"claude-3-5-sonnet-latest"},{"id":"claude-3-7-sonnet-latest"}]}`)
    models, err := ParseModels(body)
    if err != nil {
        t.Fatalf("parse models: %v", err)
    }
    if len(models) != 2 || models[0] != "claude-3-5-sonnet-latest" {
        t.Fatalf("unexpected models: %#v", models)
    }
}
```

```go
func TestGeminiModelsResponseParsesModels(t *testing.T) {
    body := []byte(`{"models":[{"name":"models/gemini-1.5-pro"},{"name":"models/gemini-1.5-flash"}]}`)
    models, err := ParseModels(body)
    if err != nil {
        t.Fatalf("parse models: %v", err)
    }
    if len(models) != 2 || models[0] != "gemini-1.5-pro" {
        t.Fatalf("unexpected models: %#v", models)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/providers/... -v`
Expected: FAIL because provider parsing functions do not exist yet.

- [ ] **Step 3: Write minimal implementation**

```go
// internal/types/types.go
package types

type UpstreamKey struct {
    ID       int64  `json:"id"`
    Name     string `json:"name"`
    Provider string `json:"provider"`
    BaseURL  string `json:"base_url"`
    APIKey   string `json:"api_key,omitempty"`
    Enabled  bool   `json:"enabled"`
}
```

```go
// internal/catalog/catalog.go
package catalog

import "sync"

type Catalog struct {
    mu     sync.RWMutex
    models map[string][]int64
}

func New() *Catalog { return &Catalog{models: map[string][]int64{}} }
func (c *Catalog) Replace(next map[string][]int64) { c.mu.Lock(); c.models = next; c.mu.Unlock() }
func (c *Catalog) KeysForModel(model string) []int64 {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return append([]int64(nil), c.models[model]...)
}
```

```go
// internal/probe/service.go
package probe

func (s *Service) ProbeKey(ctx context.Context, key types.UpstreamKey) error {
    var (
        models []string
        err    error
    )
    switch key.Provider {
    case "openai":
        models, err = s.openai.ListModels(ctx, key)
    case "anthropic":
        models, err = s.anthropic.ListModels(ctx, key)
    case "gemini":
        models, err = s.gemini.ListModels(ctx, key)
    default:
        return fmt.Errorf("unsupported provider: %s", key.Provider)
    }
    if err != nil {
        return s.state.MarkProbeFailure(ctx, key.ID, err)
    }
    if err := s.models.ReplaceForKey(ctx, key.ID, models); err != nil {
        return err
    }
    if err := s.state.MarkProbeSuccess(ctx, key.ID); err != nil {
        return err
    }
    return s.catalog.Rebuild(ctx)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/providers/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/types internal/store/keys.go internal/store/models.go internal/store/state.go internal/catalog internal/probe internal/providers
git commit -m "feat: add key registry and model probe adapters"
```

### Task 3: Implement selector, health handling, and the OpenAI-compatible gateway path

**Files:**
- Create: `internal/selector/selector.go`
- Create: `internal/usage/usage.go`
- Create: `internal/httpapi/openai_compat.go`
- Modify: `internal/httpapi/router.go`
- Test: `internal/selector/selector_test.go`
- Test: `internal/usage/usage_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestSelectorSkipsCoolingDownKeys(t *testing.T) {
    s := New()
    chosen, err := s.Select([]Candidate{
        {KeyID: 1, CoolingDown: true, Inflight: 0},
        {KeyID: 2, CoolingDown: false, Inflight: 1},
    })
    if err != nil {
        t.Fatalf("select: %v", err)
    }
    if chosen.KeyID != 2 {
        t.Fatalf("expected key 2, got %d", chosen.KeyID)
    }
}
```

```go
func TestNormalizeUsageFallback(t *testing.T) {
    usage := ParsedUsage{PromptTokens: 0, CompletionTokens: 12, TotalTokens: 12}
    if usage.TotalTokens != 12 {
        t.Fatalf("unexpected usage: %#v", usage)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/selector ./internal/usage -v`
Expected: FAIL because selector and usage helpers do not exist.

- [ ] **Step 3: Write minimal implementation**

```go
// internal/selector/selector.go
package selector

type Candidate struct {
    KeyID       int64
    CoolingDown bool
    Inflight    int
}

type Selector struct{}
func New() *Selector { return &Selector{} }
func (s *Selector) Select(candidates []Candidate) (Candidate, error) {
    filtered := make([]Candidate, 0, len(candidates))
    for _, candidate := range candidates {
        if !candidate.CoolingDown {
            filtered = append(filtered, candidate)
        }
    }
    if len(filtered) == 0 {
        return Candidate{}, errors.New("no healthy keys available")
    }
    sort.Slice(filtered, func(i, j int) bool { return filtered[i].Inflight < filtered[j].Inflight })
    return filtered[0], nil
}
```

```go
// internal/httpapi/openai_compat.go
func (h *Handler) ChatCompletions(w http.ResponseWriter, r *http.Request) {
    var req OpenAIChatRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request"})
        return
    }
    keyIDs := h.catalog.KeysForModel(req.Model)
    if len(keyIDs) == 0 {
        writeJSON(w, http.StatusBadRequest, map[string]any{"error": "no key supports requested model"})
        return
    }
    candidates, err := h.state.Candidates(r.Context(), keyIDs)
    if err != nil {
        writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
        return
    }
    selected, err := h.selector.Select(candidates)
    if err != nil {
        writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
        return
    }
    result, usage, latencyMs, callErr := h.proxy.Execute(r.Context(), selected.KeyID, req)
    _ = h.logs.Insert(r.Context(), selected.KeyID, req.Model, usage, latencyMs, callErr)
    if callErr != nil {
        _ = h.state.MarkFailure(r.Context(), selected.KeyID, callErr)
        writeJSON(w, http.StatusBadGateway, map[string]any{"error": callErr.Error()})
        return
    }
    _ = h.state.MarkSuccess(r.Context(), selected.KeyID)
    writeJSON(w, http.StatusOK, result)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/selector ./internal/usage -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/selector internal/usage internal/httpapi/openai_compat.go internal/httpapi/router.go
git commit -m "feat: add gateway selection and chat completions proxy"
```

### Task 4: Implement admin APIs, token aggregation, and health summaries

**Files:**
- Create: `internal/httpapi/admin_auth.go`
- Create: `internal/httpapi/admin_keys.go`
- Create: `internal/httpapi/admin_dashboard.go`
- Create: `internal/httpapi/admin_health.go`
- Create: `internal/store/logs.go`
- Create: `internal/store/stats.go`
- Modify: `internal/httpapi/router.go`

- [ ] **Step 1: Write the failing test**

```go
func TestTokenTotalsByModel(t *testing.T) {
    rows := []Record{
        {Model: "gpt-4o", PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
        {Model: "gpt-4o", PromptTokens: 4, CompletionTokens: 6, TotalTokens: 10},
        {Model: "claude-3-7-sonnet", PromptTokens: 7, CompletionTokens: 3, TotalTokens: 10},
    }
    stats := AggregateByModel(rows)
    if stats[0].Model != "gpt-4o" || stats[0].TotalTokens != 25 {
        t.Fatalf("unexpected stats: %#v", stats)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/store -run TokenTotalsByModel -v`
Expected: FAIL because aggregation helpers do not exist.

- [ ] **Step 3: Write minimal implementation**

```go
// internal/httpapi/admin_dashboard.go
func (h *Handler) DashboardStats(w http.ResponseWriter, r *http.Request) {
    items, summary, err := h.stats.TokenStats(r.Context())
    if err != nil {
        writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
        return
    }
    writeJSON(w, http.StatusOK, map[string]any{
        "items": items,
        "summary": summary,
        "health": h.state.Overview(r.Context()),
    })
}
```

```go
// internal/httpapi/admin_auth.go
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
    var body struct {
        Username string `json:"username"`
        Password string `json:"password"`
    }
    if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
        writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid payload"})
        return
    }
    admin, err := h.admin.Authenticate(r.Context(), body.Username, body.Password)
    if err != nil {
        writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "invalid credentials"})
        return
    }
    _ = h.sessions.Set(w, admin.Username)
    writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/store ./internal/httpapi -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/httpapi/admin_auth.go internal/httpapi/admin_keys.go internal/httpapi/admin_dashboard.go internal/httpapi/admin_health.go internal/store/logs.go internal/store/stats.go internal/httpapi/router.go
git commit -m "feat: add admin auth, key APIs, and dashboard stats"
```

### Task 5: Implement the protected web UI

**Files:**
- Create: `web/package.json`
- Create: `web/tsconfig.json`
- Create: `web/vite.config.ts`
- Create: `web/index.html`
- Create: `web/src/main.tsx`
- Create: `web/src/App.tsx`
- Create: `web/src/api.ts`
- Create: `web/src/auth.ts`
- Create: `web/src/styles.css`
- Create: `web/src/pages/LoginPage.tsx`
- Create: `web/src/pages/DashboardPage.tsx`
- Create: `web/src/pages/KeysPage.tsx`
- Create: `web/src/pages/ModelsPage.tsx`
- Create: `web/src/pages/HealthPage.tsx`
- Create: `web/src/components/Layout.tsx`
- Create: `web/src/components/StatCard.tsx`
- Test: `web/src/pages/DashboardPage.test.tsx`

- [ ] **Step 1: Write the failing test**

```tsx
import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import DashboardPage from './DashboardPage'

describe('DashboardPage', () => {
  it('renders per-model total tokens', () => {
    render(<DashboardPage data={{
      items: [{ model: 'gpt-4o', prompt_tokens: 10, completion_tokens: 5, total_tokens: 15, request_count: 1 }],
      summary: { prompt_tokens: 10, completion_tokens: 5, total_tokens: 15 }
    }} />)
    expect(screen.getByText('gpt-4o')).toBeInTheDocument()
    expect(screen.getByText('15')).toBeInTheDocument()
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd web && npm test -- DashboardPage.test.tsx`
Expected: FAIL because the frontend app has not been created.

- [ ] **Step 3: Write minimal implementation**

```tsx
// web/src/pages/DashboardPage.tsx
export default function DashboardPage({ data }: { data: DashboardResponse }) {
  return (
    <div>
      <h1>Dashboard</h1>
      <div className="stats-grid">
        <StatCard label="Prompt Tokens" value={data.summary.prompt_tokens} />
        <StatCard label="Completion Tokens" value={data.summary.completion_tokens} />
        <StatCard label="Total Tokens" value={data.summary.total_tokens} />
      </div>
      <table>
        <thead>
          <tr><th>Model</th><th>Prompt</th><th>Completion</th><th>Total</th><th>Requests</th></tr>
        </thead>
        <tbody>
          {data.items.map((item) => (
            <tr key={item.model}>
              <td>{item.model}</td>
              <td>{item.prompt_tokens}</td>
              <td>{item.completion_tokens}</td>
              <td>{item.total_tokens}</td>
              <td>{item.request_count}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd web && npm install && npm test && npm run build`
Expected: PASS tests and a generated `web/dist` directory.

- [ ] **Step 5: Commit**

```bash
git add web
git commit -m "feat: add authenticated frontend dashboard"
```

### Task 6: Wire serving, docs, Docker, and full verification

**Files:**
- Modify: `internal/app/app.go`
- Modify: `cmd/atomhub/main.go`
- Create: `Makefile`
- Create: `Dockerfile`
- Create: `docker-compose.yml`
- Create: `.env.example`
- Create: `README.md`

- [ ] **Step 1: Write the failing build command expectation**

```bash
go test ./...
cd web && npm test && npm run build && cd ..
go build ./cmd/atomhub
```

Expected: FAIL until the app is fully wired.

- [ ] **Step 2: Write minimal implementation**

```go
// internal/app/app.go
mux := http.NewServeMux()
mux.HandleFunc("/v1/chat/completions", handlers.ChatCompletions)
mux.HandleFunc("/admin/login", handlers.Login)
mux.Handle("/admin/", handlers.RequireAdmin(adminMux))
mux.Handle("/", http.FileServer(http.Dir("web/dist")))
```

```makefile
backend-test:
	go test ./...

frontend-test:
	cd web && npm test

frontend-build:
	cd web && npm run build

build: frontend-build
	go build -o bin/atomhub ./cmd/atomhub
```

- [ ] **Step 3: Run the full verification**

Run:
```bash
go test ./...
cd web && npm test && npm run build && cd ..
go build ./cmd/atomhub
```
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/app cmd/atomhub Makefile Dockerfile docker-compose.yml .env.example README.md
git commit -m "feat: ship atomhub full stack gateway"
```
