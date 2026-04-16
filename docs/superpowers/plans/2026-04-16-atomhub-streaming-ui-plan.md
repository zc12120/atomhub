# AtomHub Streaming + Key Management Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extend AtomHub with streaming chat support and richer admin key management controls while preserving the existing authenticated dashboard and provider-routing core.

**Architecture:** The backend will add SSE-compatible downstream streaming for OpenAI-style chat completions and route streams to OpenAI, Anthropic, or Gemini upstreams with best-effort protocol normalization. The admin UI will add inline enable/disable and edit flows for keys, backed by existing admin APIs plus any missing update semantics.

**Tech Stack:** Go 1.26+, net/http streaming, React + Vite + TypeScript, Vitest, existing SQLite persistence.

---

## File Structure

### Backend streaming and admin updates
- Modify: `internal/app/api_types.go`
- Modify: `internal/app/handlers_gateway.go`
- Modify: `internal/app/handlers_admin.go`
- Modify: `internal/app/app_test.go`
- Modify: `internal/store/keys.go`
- Create: `internal/app/streaming.go`
- Test: `internal/app/app_test.go`

### Frontend key-management improvements
- Modify: `web/src/api.ts`
- Modify: `web/src/pages/KeysPage.tsx`
- Modify: `web/src/styles.css`
- Test: `web/src/pages/DashboardPage.test.tsx`
- Create: `web/src/pages/KeysPage.test.tsx`

### Documentation
- Modify: `README.md`

---

### Task 1: Add backend streaming support for downstream chat completions

**Files:**
- Modify: `internal/app/api_types.go`
- Modify: `internal/app/handlers_gateway.go`
- Create: `internal/app/streaming.go`
- Modify: `internal/app/app_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestChatCompletionsStreamOpenAI(t *testing.T) {
    app := newTestAppWithStubbedOpenAIStream(t)

    body := []byte(`{"model":"gpt-4o-mini","stream":true,"messages":[{"role":"user","content":"hi"}]}`)
    req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
    req.Header.Set("Authorization", "Bearer gateway-token")
    rec := httptest.NewRecorder()

    app.Handler.ServeHTTP(rec, req)

    if rec.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
    }
    if !strings.Contains(rec.Body.String(), "data: ") {
        t.Fatalf("expected SSE payload, got %s", rec.Body.String())
    }
    if !strings.Contains(rec.Body.String(), "[DONE]") {
        t.Fatalf("expected DONE marker, got %s", rec.Body.String())
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/app -run Stream -v`
Expected: FAIL because streaming returns “not supported yet.”

- [ ] **Step 3: Write minimal implementation**

```go
// internal/app/streaming.go
func writeSSEChunk(w http.ResponseWriter, payload string) {
    _, _ = io.WriteString(w, "data: "+payload+"\n\n")
    if flusher, ok := w.(http.Flusher); ok {
        flusher.Flush()
    }
}
```

```go
// internal/app/handlers_gateway.go
if req.Stream {
    if err := a.streamChatCompletion(w, r, key, req); err != nil {
        _ = a.stateStore.MarkFailure(r.Context(), selected.KeyID, err)
        return
    }
    _ = a.stateStore.MarkSuccess(r.Context(), selected.KeyID)
    return
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/app -run Stream -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/app/api_types.go internal/app/handlers_gateway.go internal/app/streaming.go internal/app/app_test.go
git commit -m "feat: add streaming chat completions"
```

### Task 2: Add admin key enable/disable and edit support in backend + frontend

**Files:**
- Modify: `internal/app/handlers_admin.go`
- Modify: `internal/store/keys.go`
- Modify: `web/src/api.ts`
- Modify: `web/src/pages/KeysPage.tsx`
- Modify: `web/src/styles.css`
- Create: `web/src/pages/KeysPage.test.tsx`

- [ ] **Step 1: Write the failing test**

```tsx
it('shows enable/disable and edit controls for each key row', async () => {
  render(<KeysPage />)
  expect(await screen.findByRole('button', { name: /probe/i })).toBeInTheDocument()
  expect(screen.getByRole('button', { name: /disable|enable/i })).toBeInTheDocument()
  expect(screen.getByRole('button', { name: /edit/i })).toBeInTheDocument()
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd web && npm test -- KeysPage.test.tsx`
Expected: FAIL because controls do not exist.

- [ ] **Step 3: Write minimal implementation**

```tsx
<button type="button" onClick={() => openEditor(item)}>Edit</button>
<button type="button" onClick={() => toggleEnabled(item)}>
  {item.enabled ? 'Disable' : 'Enable'}
</button>
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd web && npm test -- KeysPage.test.tsx`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/app/handlers_admin.go internal/store/keys.go web/src/api.ts web/src/pages/KeysPage.tsx web/src/pages/KeysPage.test.tsx web/src/styles.css
git commit -m "feat: add key edit and enable controls"
```

### Task 3: Full verification and docs refresh

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Update README feature list**

```md
- Supports non-stream and stream chat completions
- UI supports editing and enabling/disabling keys
```

- [ ] **Step 2: Run full verification**

Run:
```bash
go test ./...
cd web && npm test && npm run build && cd ..
go build ./cmd/atomhub
```
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add README.md
git commit -m "docs: document streaming and key controls"
```
