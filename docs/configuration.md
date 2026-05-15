# Configuration and Data Model

## Configuration

The user config file is loaded from `~/.goroute/config.json`.
It configures local runtime behavior and credentials only; providers, model namespaces, and model catalogs are compiled into the binary.

The current schema has two top-level domains:

- server
- connections

Example `~/.goroute/config.json`:

```json
{
  "server": {
    "listen": ":2232",
    "auth_token": "change-me",
    "web_ui_dir": "web/dist"
  },
  "connections": [
    {
      "id": "codex-1",
      "provider_id": "cx",
      "access_token": "${ACCESS_TOKEN}",
      "refresh_token": "${REFRESH_TOKEN}",
      "name": "user@example.com"
    },
    {
      "id": "openai-1",
      "provider_id": "openai",
      "api_key": "${OPENAI_API_KEY}",
      "name": "user@example.com"
    }
  ]
}
```

Current validation requires every connection to have `id`, `provider_id`, and `name`.
`server.listen` defaults to `:2232` when omitted.
`server.auth_token` is required and is used to protect admin-only HTTP routes.
`server.web_ui_dir` defaults to `web/dist`; when that directory exists, the Go server also serves the built admin UI and SPA routes.
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
- `refresh_token` is represented in config but is not used for refresh yet.

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

### Connection config

Implemented fields:

- id
- provider_id
- name
- api_key
- access_token
- refresh_token

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
