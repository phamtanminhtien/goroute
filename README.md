# goroute

`goroute` is an OpenAI-compatible routing proxy written in Go.

Its job is to accept requests from OpenAI-style clients, resolve a client-facing model prefix into a provider/connection path, forward the request to the selected upstream connection, and return a normalized response.

This project is intended for environments where multiple upstream LLM connections need to be exposed behind one stable API surface.

## Status

Early implementation.

The repository has the first request path in place:

- `GET /v1/models`
- `POST /v1/chat/completions`
- model prefix resolution from built-in provider packages
- configured connection registry for `codex` and `openai`
- OpenAI-compatible upstream execution for non-streaming chat completions
- Codex responses execution for non-streaming and streaming chat completions
- request ID and request logging middleware

The implementation is still intentionally small. Fallback is deterministic across configured connections of the same type, but retry eligibility and richer attempt logging are not yet policy-driven. Admin APIs, UI, request history, and broader OpenAI wire compatibility are still pending.

## Core Idea

OpenAI-compatible clients are easy to integrate, but real deployments usually need more than a direct 1:1 connection to a single upstream:

- model names differ across upstreams
- credentials and base URLs vary by upstream
- fallback is often needed for reliability
- clients should not need connection-specific configuration
- routing policy should live server-side, not in every client

`goroute` centralizes that logic in one HTTP service.

## Initial Scope

The first useful version should focus on:

- OpenAI-compatible HTTP ingress
- model prefix resolution
- connection selection
- provider/connection fallback chains
- request/response passthrough with minimal normalization
- structured logging and debuggable routing behavior

Currently implemented endpoints:

- `POST /v1/chat/completions`
- `GET /v1/models`
- `GET /healthz`
- `GET /debug/requests` (admin-only, requires `Authorization: Bearer <server.auth_token>`)

## Example Usage

A local client might be configured with:

- Base URL: `http://localhost:2232/v1`
- API key: any placeholder value required by the client
- Model: `cx/gpt-5.4`

`goroute` then resolves `cx/gpt-5.4` as:

- `cx` -> system provider named `Codex`
- `gpt-5.4` -> the model passed to user-configured connections for provider `cx`

The client remains unchanged while routing policy evolves server-side.

## Configuration Direction

The user config file is loaded from `~/.goroute/config.json`.
It configures local runtime behavior and credentials only; system providers, model namespaces, and model catalogs are compiled into the binary.

Current implemented shape:

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

The current loader validates `id`, `provider_id`, and `name` for every connection. Credentials are checked by the selected adapter when a request is executed.
`server.auth_token` is required and protects admin-only backend routes such as request history.
`server.web_ui_dir` defaults to `web/dist`; when that folder exists, `goroute` also serves the built admin UI from the same server.

Connections with `provider_id: "openai"` currently target the standard OpenAI upstream only; custom OpenAI-compatible base URLs are not yet configurable.

## Admin UI

Build the web app, then start the Go server:

```bash
make web-build
make run
```

Or in one step:

```bash
make run-with-web
```

After that:

- admin UI: `http://localhost:2232/`
- admin API: `http://localhost:2232/admin/api`

During frontend development you can still use Vite separately with `make web-dev`.

## Logging

`goroute` now uses structured `zerolog` logs.

- `GOROUTE_ENV=prod` or `GOROUTE_ENV=production` emits JSON logs
- any other value, including empty, emits pretty console logs for local development

Logs include request routing and fallback metadata such as `request_id`, provider/connection fields, latency, and HTTP response status. Secrets such as bearer tokens, API keys, and request bodies are not logged.

See [Configuration and data model](./docs/configuration.md) for the fuller contract and rationale.

## Documentation

- [Design overview](./docs/design.md)
- [Configuration and data model](./docs/configuration.md)
- [Implementation plan](./docs/implementation-plan.md)

## License

See [LICENSE](./LICENSE).
