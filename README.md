# AtomHub

AtomHub is a self-hosted AI gateway and admin dashboard.

## What it does
- Manages multiple upstream OpenAI / Anthropic / Gemini API keys
- Probes which models each key supports
- Exposes a single OpenAI-compatible downstream API
- Supports both standard and streaming chat completions on the downstream API
- Load-balances requests across healthy keys that support the same model
- Cools down failing keys and retries on other eligible keys
- Persists token usage and shows per-model prompt/completion/total token totals in a web dashboard
- Protects the admin UI with a single-admin login session

## Current gateway support
### Downstream API
- `GET /v1/models`
- `POST /v1/chat/completions`

### Current limitations
- The gateway currently focuses on text chat completions
- Anthropic and Gemini streaming use a graceful OpenAI-style SSE fallback built from full upstream responses
- Admin UI supports adding, editing, enabling/disabling, probing, listing, and deleting keys; provider-specific advanced settings are intentionally minimal

## Admin UI
After login, the web UI provides:
- Dashboard: per-model token usage totals and overall totals
- Keys: create keys, edit keys, enable/disable keys, probe keys, delete keys, and view health/errors
- Models: see discovered model pools per provider
- Health: see healthy/unhealthy key counts and last errors

## Configuration
All runtime configuration is provided through environment variables:

- `ATOMHUB_HTTP_ADDR`
- `ATOMHUB_DB_PATH`
- `ATOMHUB_SESSION_SECRET`
- `ATOMHUB_SESSION_TTL`
- `ATOMHUB_ADMIN_USERNAME`
- `ATOMHUB_ADMIN_PASSWORD`
- `ATOMHUB_GATEWAY_TOKEN`

See `.env.example` for defaults.

## Local development
### Prerequisites
- Go 1.26+
- Node 22+

### Install frontend dependencies
```bash
cd web
npm install
```

### Run tests
```bash
make backend-test
make frontend-test
```

### Build everything
```bash
make build
```

### Run locally
```bash
ATOMHUB_ADMIN_PASSWORD=admin ATOMHUB_GATEWAY_TOKEN=local-token make run
```

Open `http://localhost:8080/login` and sign in with the configured admin username/password.

## Docker
### Build and run with compose
```bash
docker compose up --build -d
```

Default exposed port: `8080`

## Using the gateway
### List models
```bash
curl http://localhost:8080/v1/models \
  -H "Authorization: Bearer change-me-gateway-token"
```

### Chat completions
```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer change-me-gateway-token" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "Hello"}]
  }'
```
