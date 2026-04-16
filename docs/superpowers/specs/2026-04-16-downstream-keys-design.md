# Downstream Key Management Design

## Goal
Add a UI-managed downstream key system so clients can call AtomHub with generated keys instead of relying on one global env token.

## Scope
- Generate downstream keys in the admin UI.
- Persist only a hash + prefix; show plaintext once at creation time.
- Authenticate `/v1/*` with either a generated downstream key or the existing env `ATOMHUB_GATEWAY_TOKEN` fallback.
- Attribute request logs and usage to the downstream key that made the request.
- Provide admin CRUD/listing and basic usage stats for downstream keys.
- Add a frontend page to create, enable/disable, delete, and inspect downstream keys.

## Non-goals
- Multi-user/RBAC.
- Per-model allowlists or quotas.
- Replacing the existing admin auth/session system.

## Data model
- `downstream_keys`: id, name, token_prefix, token_hash, enabled, last_used_at, request_count, prompt_tokens, completion_tokens, total_tokens, created_at, updated_at.
- `request_logs.downstream_key_id`: nullable foreign-style reference for attribution.

## Backend behavior
- New downstream keys are generated server-side as long random bearer tokens with an `atom_` prefix.
- The database stores `sha256(token)` plus a short display prefix.
- Gateway auth checks bearer tokens in this order:
  1. exact env fallback token
  2. hashed lookup in `downstream_keys` for enabled keys
- On successful proxied requests, request logs store the downstream key id when present.
- Usage counters on the downstream key are updated from normalized token usage.

## Admin API
- `GET /admin/downstream-keys`
- `POST /admin/downstream-keys`
- `PUT /admin/downstream-keys/{id}`
- `DELETE /admin/downstream-keys/{id}`

## Frontend
- Add a `下游密钥` navigation item and page.
- Page supports create, enable/disable, delete, and usage visibility.
- After creation, the plaintext token is shown once in a highlighted success panel for copy/paste.

## Testing
- Backend tests cover token hashing/lookup, gateway auth with downstream keys, request attribution, and admin CRUD.
- Frontend tests cover page rendering, creation flow, and enabled-state updates.
