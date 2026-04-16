# Downstream Key Reveal Design

## Goal
Make downstream keys manageable like real client credentials: list them masked, reveal/copy full key on demand, and regenerate when needed.

## Chosen approach
Store each downstream key in two forms:
- `token_hash` for request authentication lookup
- encrypted plaintext for admin reveal/copy

Encryption uses a deterministic app-side secret derived from `ATOMHUB_DOWNSTREAM_KEY_SECRET`, falling back to `ATOMHUB_SESSION_SECRET` so deployment works immediately.

## Backend changes
- Add config field `DownstreamKeySecret`.
- Add encrypted storage columns to `downstream_keys`.
- Keep auth path on `token_hash` unchanged.
- Add admin endpoints:
  - `GET /admin/downstream-keys/{id}/token`
  - `POST /admin/downstream-keys/{id}/regenerate`
- Existing list responses return masked display text instead of only the prefix.
- Old rows without encrypted payload remain usable for auth but cannot be revealed; regenerate repairs them.

## Frontend changes
- Downstream key list shows masked key text.
- Row actions add:
  - 查看密钥
  - 复制密钥
  - 重新生成
- Reveal fetches the full key and displays it inline for that row.
- Copy uses the revealed key when available; otherwise it fetches once before copying.

## Safety notes
- Full key is never returned in list APIs.
- Plaintext only appears in create/reveal/regenerate responses.
- Regeneration rotates both the auth hash and encrypted plaintext.
