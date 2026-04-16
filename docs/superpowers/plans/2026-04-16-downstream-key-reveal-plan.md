# Downstream Key Reveal Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add masked display, reveal/copy, and regenerate support for downstream keys while preserving hash-based auth.

**Architecture:** Extend downstream key persistence with encrypted plaintext plus unchanged hash lookup, then expose narrow admin endpoints for reveal/regenerate and update the React page to use them.

**Tech Stack:** Go, SQLite, React, Vitest.

---

### Task 1: Backend encryption and APIs

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/types/types.go`
- Modify: `internal/store/migrate.go`
- Modify: `internal/store/downstream_keys.go`
- Modify: `internal/app/api_types.go`
- Modify: `internal/app/app.go`
- Modify: `internal/app/handlers_admin.go`
- Test: `internal/app/downstream_keys_test.go`
- Test: `internal/auth/downstream_test.go`

- [ ] Add encrypted plaintext storage, reveal/regenerate APIs, and backend tests.

### Task 2: Frontend masked/reveal/copy UX

**Files:**
- Modify: `web/src/api.ts`
- Modify: `web/src/pages/DownstreamKeysPage.tsx`
- Modify: `web/src/pages/DownstreamKeysPage.test.tsx`
- Modify: `web/src/styles.css`

- [ ] Update the downstream-key page to show masked values and support reveal/copy/regenerate.

### Task 3: Verification and deploy docs

**Files:**
- Modify: `README.md`

- [ ] Update docs, run `go test ./...`, `go build ./cmd/atomhub`, `cd web && npm test`, `cd web && npm run build`, then commit.
