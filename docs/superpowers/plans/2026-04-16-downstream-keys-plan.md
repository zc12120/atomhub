# Downstream Key Management Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add UI-managed downstream bearer keys that authenticate `/v1/*`, persist usage, and can be managed from the AtomHub admin frontend.

**Architecture:** Add a dedicated downstream key store plus request context plumbing in the Go backend, then expose CRUD endpoints and a small admin page in the React frontend. Keep the existing env gateway token as a fallback super-token so existing clients do not break.

**Tech Stack:** Go, net/http, SQLite, React, Vite, Vitest.

---

### Task 1: Backend storage and auth

**Files:**
- Create: `internal/auth/downstream.go`
- Create: `internal/store/downstream_keys.go`
- Modify: `internal/store/migrate.go`
- Modify: `internal/types/types.go`
- Modify: `internal/app/handlers_gateway.go`
- Modify: `internal/app/routes.go`
- Test: `internal/app/downstream_keys_test.go`

- [ ] Add downstream key schema, token generation/hash helpers, gateway auth context plumbing, and tests for downstream bearer auth plus env-token fallback.

### Task 2: Attribution and admin APIs

**Files:**
- Modify: `internal/store/logs.go`
- Modify: `internal/store/stats.go`
- Modify: `internal/app/api_types.go`
- Modify: `internal/app/app.go`
- Modify: `internal/app/handlers_admin.go`
- Modify: `internal/app/requests_test.go`
- Test: `internal/app/downstream_admin_test.go`

- [ ] Add downstream-key CRUD/list responses, usage counters, request log attribution, and admin tests.

### Task 3: Frontend management page

**Files:**
- Modify: `web/src/api.ts`
- Modify: `web/src/App.tsx`
- Modify: `web/src/components/Layout.tsx`
- Create: `web/src/pages/DownstreamKeysPage.tsx`
- Create: `web/src/pages/DownstreamKeysPage.test.tsx`

- [ ] Build the downstream key management page and wire it into navigation plus admin API bindings.

### Task 4: Verification and docs

**Files:**
- Modify: `README.md`

- [ ] Document the new downstream key workflow, run `go test ./...`, `cd web && npm test`, `cd web && npm run build`, then commit.
