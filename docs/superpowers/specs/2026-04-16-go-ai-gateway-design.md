# Lightweight AI Gateway Design

**Date:** 2026-04-16
**Status:** Approved for planning
**Audience:** Single self-hosted administrator

## Goal
Build a complete self-hosted project that provides a single OpenAI-compatible downstream API and web admin UI for managing multiple upstream AI keys, probing supported models, load balancing requests across keys that support the same model, converting between protocols for OpenAI/Anthropic/Gemini upstreams, and showing persistent token usage statistics in a dashboard.

## Product Scope
The project will include:
- A Go backend service
- A web frontend with single-admin authentication
- SQLite persistence
- OpenAI-compatible downstream API
- Upstream adapters for OpenAI, Anthropic, and Gemini
- Model probing and model-to-key pool management
- Same-model load balancing, cooldown, and retry
- Persistent request usage logs and dashboard statistics

The first version will not prioritize:
- Multi-user or RBAC
- Distributed deployment
- Complex audit/compliance workflows
- Full tracing platform features
- Very advanced routing policies beyond same-model balancing and health-based failover

## User-Facing Capabilities
The finished system should let the administrator:
1. Log in to a protected web UI with a single admin account.
2. Add, edit, enable, disable, and delete upstream keys.
3. Mark each upstream key as OpenAI, Anthropic, or Gemini compatible.
4. Run model probes and see which models each key supports.
5. Expose one OpenAI-compatible downstream endpoint for clients.
6. Route a request for a model to any healthy key that supports that model.
7. Automatically retry and cool down failing keys.
8. View per-model prompt/completion/total token usage and global totals.
9. View recent key health, errors, cooldown state, and request activity.
10. Restart the service without losing keys, probe data, or token statistics.

## Architecture
### Backend
A single Go binary will own four concerns:
- Control plane: admin auth, key CRUD, probe operations, dashboard APIs
- Data plane: downstream OpenAI-compatible request handling
- Adapter layer: conversion from downstream OpenAI-compatible payloads to upstream provider-specific payloads and responses
- Persistence layer: SQLite-backed storage and in-memory hot-path indexes

### Frontend
A lightweight React frontend will be built and served by the backend as static assets. The UI will focus on:
- Login page
- Key management page
- Models/probe page
- Dashboard page
- Health/status page

### Persistence
SQLite will store durable state:
- Admin auth configuration/session backing data as needed
- Upstream keys and metadata
- Probe results (key -> supported models)
- Health/cooldown state
- Request usage logs for dashboard aggregation

In-memory indexes will accelerate hot-path routing:
- model -> eligible keys
- key -> health/cooldown
- key -> inflight count

## Core Components
### 1. Admin Authentication
Single-admin authentication for the web UI and admin API.
- Session/cookie-based login for the frontend
- Protected admin endpoints
- No multi-user roles in v1

### 2. Upstream Key Registry
Stores and manages all upstream credentials.
Each key record includes:
- Provider type
- Base URL if needed
- Secret/token
- Enabled/disabled status
- Optional label/notes

### 3. Probe Engine
Detects supported models per key.
- Manual probe from UI/API
- Startup refresh
- Optional scheduled reprobe
- Persist last probe time, last error, and discovered models

### 4. Model Catalog
Derived mapping of models to supporting keys.
- Built from persisted probe data
- Refreshed after probe updates or key changes
- Used directly by the request router

### 5. Protocol Adapters
The public API is OpenAI-compatible.
Adapters will transform requests/responses for:
- OpenAI upstream
- Anthropic upstream
- Gemini upstream

Initial focus is text generation and token usage accounting. Support will prioritize the request shapes needed for practical chat-style usage first.

### 6. Selector / Load Balancer
For each requested model:
- Filter keys that support the model
- Exclude disabled or cooling-down keys
- Prefer healthy keys
- Choose among eligible keys using a simple, deterministic, lightweight strategy such as least-inflight or weighted round-robin

### 7. Failure Handling
The system will classify upstream failures and respond accordingly:
- 401/403: mark key unhealthy or disabled candidate
- 429: temporary cooldown
- timeout/5xx: retry another eligible key and apply cooldown
- repeated failures: extend cooldown or mark degraded

### 8. Usage Logging and Dashboard Aggregation
Each completed request should persist:
- model
- key used
- prompt tokens
- completion tokens
- total tokens
- status
- latency
- timestamp

Dashboard APIs will aggregate:
- Per-model token totals
- Global token totals
- Request counts
- Recent error/failure summaries

## Data Model
The minimum durable tables are:
- `admin_users` or equivalent minimal auth/session support
- `upstream_keys`
- `key_models`
- `key_state`
- `request_logs`

### `upstream_keys`
Stores key identity and configuration.
### `key_models`
Stores discovered model support per key.
### `key_state`
Stores health, cooldown, and recent failure information.
### `request_logs`
Stores token usage and request outcomes for dashboard queries.

## Request Flow
1. Client sends an OpenAI-compatible request to the gateway.
2. Gateway authenticates downstream request if configured.
3. Requested model is resolved against the in-memory catalog.
4. Eligible keys are filtered by health and cooldown.
5. Selector chooses one key.
6. Adapter converts request for the chosen upstream provider.
7. Proxy executes the upstream request.
8. Adapter normalizes response back to downstream format.
9. Usage is extracted and persisted.
10. Health/cooldown state is updated.
11. Response is returned to client.

## Error Handling
The system will return clear errors when:
- No key supports a requested model
- All eligible keys are cooling down or unhealthy
- Downstream request is malformed
- Admin auth fails

Operational protections:
- Retry only across same-model eligible keys
- Avoid infinite retry loops
- Persist cooldown state so restarts do not immediately reset bad keys into service if that is unsafe

## Testing Strategy
The project should include:
- Unit tests for selector, cooldown logic, token aggregation, and adapters
- Integration tests for key probing and routing
- End-to-end tests for admin login, key management, and dashboard token views
- Manual verification path for a locally running full stack

## Deployment Shape
The target deployment is a single self-hosted project that can run with:
- one Go service process
- one SQLite database file
- one built frontend bundle

Docker support should be included so the service can be deployed on a VPS easily.

## Out-of-Scope for Initial Delivery
- Distributed multi-node coordination
- Provider-specific advanced features not needed for basic chat routing
- Full observability stack (traces, spans, exporters)
- Rich permission system
- Large enterprise admin workflows

## Success Criteria
The first full project is successful when:
1. A single admin can log into the UI.
2. Keys can be managed and persisted.
3. Model probe results are visible and persisted.
4. A client can call one OpenAI-compatible endpoint.
5. Requests can be served through OpenAI, Anthropic, or Gemini upstreams via conversion.
6. Same-model keys are balanced and failed keys are cooled down.
7. The dashboard shows per-model prompt/completion/total tokens and global totals.
8. Restarting the service preserves operational state and usage data.
