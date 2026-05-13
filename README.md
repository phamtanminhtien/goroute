# goroute

`goroute` is an OpenAI-compatible routing proxy written in Go.

Its job is to accept requests from OpenAI-style clients, resolve a client-facing model prefix into a driver/provider path, forward the request to the selected provider, and return a normalized response.

This project is intended for environments where multiple upstream LLM providers need to be exposed behind one stable API surface.

## Status

Early bootstrap.

The repository currently defines project direction and expected behavior. Implementation is still minimal / not yet present.

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

Initial target endpoints:

- `POST /v1/chat/completions`
- `GET /v1/models`

## Example Usage

A local client might be configured with:

- Base URL: `http://localhost:2232/v1`
- API key: proxy token
- Model: `cx/gpt-5.4`

`goroute` then resolves `cx/gpt-5.4` as:

- `cx` -> system driver named `Codex`
- `gpt-5.4` -> the model passed to user-configured providers with `"type": "codex"`

The client remains unchanged while routing policy evolves server-side.

## Configuration Direction

The user config file is expected at `~/.goroute/config.json`.
It should configure local runtime behavior and credentials only; it should not define drivers, model namespaces, or model catalogs.

Illustrative example:

```json
{
  "server": {
    "listen": ":2232",
    "auth_token": "change-me"
  },
  "providers": [
    {
      "type": "codex",
      "access_token": "${ACCESS_TOKEN}",
      "refresh_token": "${REFRESH_TOKEN}",
      "name": "[EMAIL_ADDRESS]"
    },
    {
      "type": "openai",
      "api_key": "${OPENAI_API_KEY}",
      "name": "[EMAIL_ADDRESS]"
    }
  ]
}
```

This example is descriptive, not a committed spec.

## Documentation

- [Design overview](./docs/design.md)
- [Configuration and data model](./docs/configuration.md)
- [Implementation plan](./docs/implementation-plan.md)

## License

See [LICENSE](./LICENSE).
