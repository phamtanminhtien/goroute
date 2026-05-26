# Configuration and Data Model

## Configuration

The user config file is loaded from `~/.goroute/config.json`.
It configures local runtime behavior only; providers, model namespaces, and model catalogs are compiled into the binary, while connection records are stored in SQLite.

The current schema has three top-level domains:

- server
- llmLogging
- rtk

Example `~/.goroute/config.json`:

```json
{
  "server": {
    "listen": ":2232",
    "auth_token": "change-me",
    "web_ui_dir": "web/dist"
  },
  "llmLogging": false,
  "rtk": {
    "enabled": true
  }
}
```

`server.listen` defaults to `:2232` when omitted.
`server.auth_token` is required and is used to protect admin-only HTTP routes.
`server.web_ui_dir` defaults to `web/dist`; when that directory exists, the Go server also serves the built admin UI and SPA routes.
`llmLogging` controls optional persisted LLM request detail:

- when `llmLogging` is omitted, both `flow` and `thirdParty` default to disabled
- enable optional LLM log stores with `llmLogging: { "flow": true, "thirdParty": true }`
- `llmLogging: false` disables both optional log stores
- `llmLogging: {}` also disables both optional log stores
- `llmLogging.flow` controls persistence to `ai_request_flows`
- `llmLogging.thirdParty` controls persistence to `third_party_request_logs`
- `ai_request_runs` is always persisted and is not controlled by `llmLogging`

`rtk` controls optional deterministic request compression:

- when `rtk` is omitted, it defaults to enabled
- `rtk: false` disables RTK compression
- `rtk: true` enables RTK compression
- `rtk: { "enabled": true }` also enables RTK compression
- RTK only compresses large machine-generated user/tool text before upstream dispatch
- RTK does not rewrite `system`, `assistant`, or `instructions` content
- RTK diagnostics are persisted to `rtk_records` when request logging completes

Connections are persisted in `~/.goroute/goroute.db`.
Connection credentials are validated lazily by the selected adapter during request execution.
Request diagnostics are persisted in the same SQLite database. `ai_request_runs` is always written, while detailed flow logs, third-party upstream logs, and RTK diagnostics depend on runtime settings and request behavior.

## Logging Environment

Logging format is controlled by the runtime environment variable `GOROUTE_ENV`.

- `GOROUTE_ENV=prod` or `GOROUTE_ENV=production` emits JSON logs
- `GOROUTE_ENV=dev`, `GOROUTE_ENV=local`, empty, or any other value emits pretty console logs

Structured logs include request, routing, connection fallback, and admin connection-management metadata.
Sensitive values such as `api_key`, `access_token`, `refresh_token`, bearer tokens, and request bodies are not logged.

Current connection credential behavior:

- `codex` uses `access_token`, falling back to `api_key` if present.
- `openai` uses `api_key`, falling back to `access_token` if present.
- `anthropic` uses `api_key`.
- `refresh_token` is stored with the connection record and is used by the Codex provider to refresh access tokens when needed.
- `token_type` and `expires_in` are persisted with OAuth connections when returned by the provider.

## Admin API

Admin HTTP routes are protected by `Authorization: Bearer <server.auth_token>`.

Current admin routes:

- `GET /admin/api/providers`
- `POST /admin/api/providers/{id}/oauth-url`
- `POST /admin/api/providers/{id}/test`
- `GET /admin/api/connections`
- `POST /admin/api/connections`
- `GET /admin/api/connections/{id}`
- `PUT /admin/api/connections/{id}`
- `DELETE /admin/api/connections/{id}`
- `GET /admin/api/connections/{id}/usage`
- `POST /admin/api/connections/oauth`
- `GET /admin/api/settings`
- `PUT /admin/api/settings`
- `GET /admin/api/analytics/usage/summary`
- `GET /admin/api/analytics/usage/timeseries`
- `GET /admin/api/analytics/usage/provider-breakdown`
- `GET /admin/api/analytics/usage/recent-requests`
- `GET /admin/api/logs/stream`

`GET /admin/api/settings` returns a normalized payload for admin UI use:

```json
{
  "server": {
    "listen": ":2232",
    "web_ui_dir": "web/dist"
  },
  "llmLogging": {
    "enabled": {
      "flow": true,
      "thirdParty": true
    }
  },
  "rtk": {
    "enabled": false
  }
}
```

`PUT /admin/api/settings` accepts the same normalized `llmLogging.enabled` booleans plus `rtk.enabled`, then applies them immediately to new requests after saving `config.json`.

`POST /admin/api/providers/{id}/test` validates that a configured provider can execute a small model test through the current connection registry.

The analytics usage routes summarize persisted `ai_request_runs` records for the admin UI:

- `summary` returns aggregate request, token, cost, latency, and status totals for a time window
- `timeseries` buckets usage over time
- `provider-breakdown` groups usage by provider/model
- `recent-requests` returns recent request rows for inspection

`GET /admin/api/logs/stream` streams live server logs for the admin console page. It is an operational stream, not a persisted log-history API.

### Quota Tracker lookup

`GET /admin/api/connections/{id}/usage` currently has special support for Codex connections.

This usage data is not persisted in a database, cache table, or the user config file.
The server fetches it live from the Codex upstream when the admin route is called, normalizes it in memory, and returns it immediately.
Nothing from the normalized quota payload is stored back into `~/.goroute/config.json` or `~/.goroute/goroute.db`.

The Codex adapter calls:

```text
GET https://chatgpt.com/backend-api/wham/usage
Authorization: Bearer <connection access token>
Accept: application/json
```

One upstream response shape the adapter explicitly expects to handle is:

```json
{
  "plan_type": "plus",
  "rate_limit": {
    "limit_reached": false,
    "primary_window": {
      "used_percent": 42,
      "reset_at": 1747404000
    },
    "secondary_window": {
      "used_percent": 68,
      "reset_at": 1747922400
    }
  },
  "code_review_rate_limit": {
    "limit_reached": false,
    "primary_window": {
      "used_percent": 15,
      "reset_at": 1747404000
    },
    "secondary_window": {
      "used_percent": 27,
      "reset_at": 1747922400
    }
  }
}
```

The response is normalized before it reaches the UI:

- `plan` comes from `plan_type`, then `summary.plan`, then falls back to `unknown`
- normal quota comes from `rate_limit`, `rate_limits`, or `rate_limits_by_limit_id.codex`
- review quota comes from `code_review_rate_limit`, `review_rate_limit`, `rate_limits_by_limit_id.code_review`, `rate_limits_by_limit_id.codex_review`, or the first `additional_rate_limits[]` entry whose `id` contains `review`
- `primary_window` or `primary` maps to `session`
- `secondary_window` or `secondary` maps to `weekly`

Normalized response shape:

```json
{
  "plan": "plus",
  "limitReached": false,
  "reviewLimitReached": false,
  "quotas": {
    "session": {
      "used": 42,
      "total": 100,
      "remaining": 58,
      "resetAt": "2026-05-16T10:00:00.000Z",
      "unlimited": false
    },
    "weekly": {
      "used": 68,
      "total": 100,
      "remaining": 32,
      "resetAt": "2026-05-22T10:00:00.000Z",
      "unlimited": false
    },
    "review_session": {
      "used": 15,
      "total": 100,
      "remaining": 85,
      "resetAt": "2026-05-16T10:00:00.000Z",
      "unlimited": false
    },
    "review_weekly": {
      "used": 27,
      "total": 100,
      "remaining": 73,
      "resetAt": "2026-05-22T10:00:00.000Z",
      "unlimited": false
    }
  }
}
```

So the admin route response shape for `GET /admin/api/connections/{id}/usage` is effectively one of:

- a normalized quota payload with `plan`, `limitReached`, `reviewLimitReached`, and optional `quotas`
- or a message-only payload when the upstream usage API is temporarily unavailable

`reset_at` is normalized broadly:

- Unix seconds are converted to milliseconds first
- Unix milliseconds are used directly
- strings are parsed and re-emitted as UTC ISO timestamps

If the upstream usage API responds with a non-2xx status, the admin route still returns `200 OK` with:

```json
{
  "message": "Codex connected. Usage API temporarily unavailable (STATUS)."
}
```

This lets the UI keep the connection visible even when quota lookup is temporarily unavailable.

## Custom Upstream Base URL Direction

The current implementation uses the built-in OpenAI base URL (`https://api.openai.com`) for connections with `provider_id: "openai"`.
There is no config field yet for overriding this per connection.

That is an intentional bootstrap constraint for now:

- keep the config contract small while real execution settles
- avoid introducing an underspecified field before fallback and observability are in place
- leave room for other OpenAI-compatible upstreams without prematurely hard-coding policy

The likely future direction is a per-connection optional field such as `base_url` on OpenAI-compatible connections, rather than a global setting.
That would preserve the existing connection-centric config shape and allow multiple OpenAI-compatible accounts or vendors side by side.

Until that lands, `provider_id: "openai"` should be read as “the standard OpenAI upstream” rather than “any OpenAI-compatible endpoint.”
Likewise, `provider_id: "anthropic"` should be read as “the standard Anthropic upstream” rather than a configurable Anthropic-compatible endpoint.

## System Providers

System providers are compiled into the binary as provider packages.

Current built-in providers:

- `cx`
- `openai`
- `anthropic`

Model IDs are prefixed with the provider ID. Examples:

- `cx/gpt-5.4`
- `cx/gpt-5.3-codex`
- `openai/gpt-4.1`
- `openai/o4-mini`
- `anthropic/claude-sonnet-4-5`
- `anthropic/claude-opus-4-1`
- `anthropic/claude-haiku-4-5`

## Data Model

### Connection record

Implemented fields:

- id
- provider_id
- name
- api_key
- access_token
- refresh_token
- token_type
- expires_in

### RTK record

Implemented fields:

- id
- run_id
- request_id
- applied
- bytes_before
- bytes_after
- saved_bytes
- saved_percent
- filter_chain
- hit_count
- field_count

### Request diagnostics

Implemented persisted request diagnostics include:

- `ai_request_runs` for one row per completed request
- `ai_request_flows` for reconstructed request/response flow details when enabled
- `third_party_request_logs` for upstream request/response logging when enabled
- `rtk_records` for compression diagnostics when RTK evaluates a request

The admin analytics APIs read from these request-run records; Codex quota lookup remains live upstream data and is not written to the analytics tables.

### System provider definition

Implemented fields:

- provider ID / client-facing prefix, such as `cx`
- display name
- auth type, such as `oauth` or `api_key`
- default model
- supported models
- optional metadata

### Resolved execution target

At request time, routing produces:

- client-facing prefix
- requested upstream model
- provider ID and name

Connection selection and fallback currently happen in the connection registry after resolution.
Fallback attempt index and request-scoped timeout metadata are not yet represented in the resolved target.
