# Configuration and Data Model

## Configuration

The user config file is loaded from `~/.goroute/config.json`.
It configures local runtime behavior only; providers, model namespaces, and model catalogs are compiled into the binary, while connection records are stored in SQLite.

The current schema has one top-level domain:

- server

Example `~/.goroute/config.json`:

```json
{
  "server": {
    "listen": ":2232",
    "auth_token": "change-me",
    "web_ui_dir": "web/dist"
  }
}
```

`server.listen` defaults to `:2232` when omitted.
`server.auth_token` is required and is used to protect admin-only HTTP routes.
`server.web_ui_dir` defaults to `web/dist`; when that directory exists, the Go server also serves the built admin UI and SPA routes.
Connections and request-attempt history are persisted in `~/.goroute/goroute.db`.
Connection credentials are validated lazily by the selected adapter during request execution.

## Logging Environment

Logging format is controlled by the runtime environment variable `GOROUTE_ENV`.

- `GOROUTE_ENV=prod` or `GOROUTE_ENV=production` emits JSON logs
- `GOROUTE_ENV=dev`, `GOROUTE_ENV=local`, empty, or any other value emits pretty console logs

Structured logs include request, routing, connection fallback, and admin connection-management metadata.
Sensitive values such as `api_key`, `access_token`, `refresh_token`, bearer tokens, and request bodies are not logged.

Current connection credential behavior:

- `codex` uses `access_token`, falling back to `api_key` if present.
- `openai` uses `api_key`, falling back to `access_token` if present.
- `refresh_token` is stored with the connection record and is used by the Codex provider to refresh access tokens when needed.
- `token_type` and `expires_in` are persisted with OAuth connections when returned by the provider.

## Admin API

Admin HTTP routes are protected by `Authorization: Bearer <server.auth_token>`.

Current admin routes:

- `GET /admin/api/providers`
- `POST /admin/api/providers/{id}/oauth-url`
- `GET /admin/api/connections`
- `POST /admin/api/connections`
- `GET /admin/api/connections/{id}`
- `PUT /admin/api/connections/{id}`
- `DELETE /admin/api/connections/{id}`
- `GET /admin/api/connections/{id}/usage`
- `POST /admin/api/connections/oauth`

### Codex usage lookup

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

## Custom OpenAI-Compatible Base URL Direction

The current implementation uses the built-in OpenAI base URL (`https://api.openai.com`) for connections with `provider_id: "openai"`.
There is no config field yet for overriding this per connection.

That is an intentional bootstrap constraint for now:

- keep the config contract small while real execution settles
- avoid introducing an underspecified field before fallback and observability are in place
- leave room for other OpenAI-compatible upstreams without prematurely hard-coding policy

The likely future direction is a per-connection optional field such as `base_url` on OpenAI-compatible connections, rather than a global setting.
That would preserve the existing connection-centric config shape and allow multiple OpenAI-compatible accounts or vendors side by side.

Until that lands, `provider_id: "openai"` should be read as “the standard OpenAI upstream” rather than “any OpenAI-compatible endpoint.”

## System Providers

System providers are compiled into the binary as provider packages.

Current built-in providers:

- `cx`
- `openai`

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
