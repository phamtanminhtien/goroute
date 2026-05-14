# goroute

`goroute` is an OpenAI-compatible routing proxy written in Go.

Its job is to accept requests from OpenAI-style clients, resolve a client-facing model prefix into a driver/provider path, forward the request to the selected provider, and return a normalized response.

This project is intended for environments where multiple upstream LLM providers need to be exposed behind one stable API surface.

## Status

Early implementation.

The repository has the first request path in place:

- `GET /v1/models`
- `POST /v1/chat/completions`
- model prefix resolution from `data/system-drivers.json`
- configured provider registry for `codex` and `openai`
- OpenAI-compatible upstream execution for non-streaming chat completions
- Codex responses execution for non-streaming and streaming chat completions
- request ID and request logging middleware

The implementation is still intentionally small. Fallback is deterministic across configured providers of the same type, but retry eligibility and richer attempt logging are not yet policy-driven. Admin APIs, UI, request history, and broader OpenAI wire compatibility are still pending.

## Core Idea

OpenAI-compatible clients are easy to integrate, but real deployments usually need more than a direct 1:1 connection to a single provider:

- model names differ across providers
- credentials and base URLs vary by upstream
- fallback is often needed for reliability
- clients should not need provider-specific configuration
- routing policy should live server-side, not in every client

`goroute` centralizes that logic in one HTTP service.

## Initial Scope

The first useful version should focus on:

- OpenAI-compatible HTTP ingress
- model prefix resolution
- provider selection
- driver/provider fallback chains
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

- `cx` -> system driver named `Codex`
- `gpt-5.4` -> the model passed to user-configured providers with `"type": "codex"`

The client remains unchanged while routing policy evolves server-side.

## Configuration Direction

The user config file is loaded from `~/.goroute/config.json`.
It configures local runtime behavior and credentials only; it does not define drivers, model namespaces, or model catalogs.

Current implemented shape:

```json
{
  "server": {
    "listen": ":2232",
    "auth_token": "change-me"
  },
  "providers": [
    {
      "id": "codex-1",
      "type": "codex",
      "access_token": "${ACCESS_TOKEN}",
      "refresh_token": "${REFRESH_TOKEN}",
      "name": "user@example.com"
    },
    {
      "id": "openai-1",
      "type": "openai",
      "api_key": "${OPENAI_API_KEY}",
      "name": "user@example.com"
    }
  ]
}
```

The current loader validates `id`, `type`, and `name` for every provider. Credentials are checked by the provider adapter when a request is executed.
`server.auth_token` is required and protects admin-only backend routes such as request history.

`type: "openai"` currently targets the standard OpenAI upstream only; custom OpenAI-compatible base URLs are not yet configurable.

See [Configuration and data model](./docs/configuration.md) for the fuller contract and rationale.

## Documentation

- [Design overview](./docs/design.md)
- [Configuration and data model](./docs/configuration.md)
- [Implementation plan](./docs/implementation-plan.md)

## License

See [LICENSE](./LICENSE).
