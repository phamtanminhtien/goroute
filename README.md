# goroute

`goroute` is an OpenAI-compatible routing proxy written in Go.

Its job is to accept requests from OpenAI-style clients, resolve a client-facing model prefix into a provider/connection path, forward the request to the selected upstream connection, and return a normalized response.

This project is intended for environments where multiple upstream LLM connections need to be exposed behind one stable API surface.

## Status

Early implementation.

The repository has the first request path in place:

- `GET /v1/models`
- `POST /v1/chat/completions`
- `POST /v1/responses`
- model prefix resolution from built-in provider packages
- configured connection registry for `codex` and `openai`
- OpenAI-compatible upstream execution for non-streaming chat completions
- OpenAI-compatible upstream execution for sync and streaming responses
- Codex responses execution for non-streaming and streaming chat completions
- Codex responses passthrough for streaming and sync reconstruction for `/v1/responses`
- admin APIs and UI for provider/connection management
- Quota Tracker lookup from the admin API for Codex connections
- request ID and request logging middleware

The implementation is still intentionally small. Fallback is deterministic across configured connections of the same type, but retry eligibility and richer attempt logging are not yet policy-driven. Broader OpenAI wire compatibility is still pending.

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
- `POST /v1/responses`
- `GET /v1/models`
- `GET /healthz`
- `GET /admin/api/providers` (admin-only)
- `POST /admin/api/providers/{id}/oauth-url` (admin-only)
- `GET /admin/api/connections` (admin-only)
- `POST /admin/api/connections` (admin-only)
- `GET /admin/api/connections/{id}` (admin-only)
- `PUT /admin/api/connections/{id}` (admin-only)
- `DELETE /admin/api/connections/{id}` (admin-only)
- `GET /admin/api/connections/{id}/usage` (admin-only, Quota Tracker lookup for Codex connections)
- `POST /admin/api/connections/oauth` (admin-only)

## Example Usage

A local client might be configured with:

- Base URL: `http://localhost:2232/v1`
- API key: any placeholder value required by the client
- Model: `cx/gpt-5.4`

`goroute` then resolves `cx/gpt-5.4` as:

- `cx` -> system provider named `Codex`
- `gpt-5.4` -> the model passed to user-configured connections for provider `cx`

The client remains unchanged while routing policy evolves server-side.

`/v1/responses` follows the same prefixed-model routing behavior. For OpenAI connections the request and response body are passed through in Responses shape. For Codex connections streaming is passed through as Responses SSE and sync responses are reconstructed from the upstream event stream.

`/v1/chat/completions` also accepts mixed text+image user content in OpenAI-compatible form:

```json
{
  "model": "cx/gpt-5.4",
  "messages": [
    {
      "role": "user",
      "content": [
        { "type": "text", "text": "What is in this image?" },
        {
          "type": "image_url",
          "image_url": {
            "url": "https://example.com/cat.png",
            "detail": "high"
          }
        }
      ]
    }
  ]
}
```

Data URLs are supported too, for example `"url": "data:image/png;base64,..."`.

`/v1/responses` keeps the corresponding Responses shape with `input_image` parts:

```json
{
  "model": "cx/gpt-5.4",
  "input": [
    {
      "type": "message",
      "role": "user",
      "content": [
        { "type": "input_text", "text": "What is in this image?" },
        { "type": "input_image", "image_url": "https://example.com/cat.png", "detail": "high" }
      ]
    }
  ]
}
```

This phase does not add multipart upload or server-side image storage; clients must send a remote URL or data URL inside the JSON payload.

## Configuration Direction

The user config file is loaded from `~/.goroute/config.json`.
It configures local runtime behavior only; system providers, model namespaces, model catalogs, and persisted connections are handled outside the config file.

Current implemented shape:

```json
{
  "server": {
    "listen": ":2232",
    "auth_token": "change-me",
    "web_ui_dir": "web/dist"
  },
  "llmLogging": {
    "flow": true,
    "thirdParty": true
  },
  "rtk": {
    "enabled": true
  }
}
```

`server.auth_token` is required and protects admin-only backend routes.
`server.web_ui_dir` defaults to `web/dist`; when that folder exists, `goroute` also serves the built admin UI from the same server.
When `llmLogging` is omitted, both `flow` and `thirdParty` default to enabled. `llmLogging: false` or `llmLogging: {}` disables both optional LLM log stores while still keeping `ai_request_runs`.
`rtk` defaults to enabled. When enabled, it applies deterministic compression to large machine-generated user/tool request text before upstream dispatch and persists diagnostics in `rtk_records`.
Connections are persisted in `~/.goroute/goroute.db` and are created through the admin API or UI.

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

The Quota Tracker page fetches Codex usage from `GET /admin/api/connections/{id}/usage` and shows normalized quota buckets for:

- `session`
- `weekly`
- `review_session`
- `review_weekly`

During frontend development you can still use Vite separately with `make web-dev`.

## Docker

You can run `goroute` with Docker Compose using either a published image from GitHub Container Registry or a local build from this repository.

Start from a published image:

```bash
docker compose up
```

Build locally from the current source:

```bash
docker compose up --build
```

Override the image tag explicitly when needed:

```bash
GOROUTE_IMAGE=ghcr.io/phamtanminhtien/goroute:latest docker compose up
```

The Compose setup exposes `localhost:2232` and persists `~/.goroute` container data in the named Docker volume `goroute-data`, including `config.json` and `goroute.db`.

## Logging

`goroute` now uses structured `zerolog` logs.

- `GOROUTE_ENV=prod` or `GOROUTE_ENV=production` emits JSON logs
- any other value, including empty, emits pretty console logs for local development

Logs include request routing and fallback metadata such as `request_id`, provider/connection fields, latency, and HTTP response status. Secrets such as bearer tokens, API keys, and request bodies are not logged.

See [Configuration and data model](./docs/configuration.md) for the fuller contract and rationale.

## Documentation

- [Design overview](./docs/design.md)
- [Configuration and data model](./docs/configuration.md)
- [RTK request compression](./docs/rtk.md)
- [Implementation plan](./docs/implementation-plan.md)

## License

See [LICENSE](./LICENSE).
