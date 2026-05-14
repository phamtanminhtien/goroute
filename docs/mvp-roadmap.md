# MVP Roadmap

This file tracks the current implementation state and the next work needed to turn the first request path into a more reliable MVP.

## Current State

Implemented now:

- OpenAI-compatible HTTP surface for:
  - `GET /v1/models`
  - `POST /v1/chat/completions`
- `GET /healthz`
- bearer-token auth middleware reserved for future admin APIs
- request ID and request logging middleware
- system driver catalog loading from `data/system-drivers.json`
- model prefix resolution such as `cx/gpt-5.4`
- default model resolution when the client sends only a driver prefix
- provider execution interface near `internal/usecase/chatcompletion`
- provider registry that selects providers by resolved provider type
- deterministic fallback across multiple configured providers of the same type
- configured provider wiring at app startup
- OpenAI-compatible upstream adapter for non-streaming chat completions
- Codex adapter for non-streaming and streaming chat completions
- normalized upstream error wrapper mapped to a gateway response by the HTTP layer
- config defaulting before validation
- basic tests for routing, HTTP contract shape, provider registry behavior, config validation, and Codex adapter mapping

Not implemented yet:

- explicit retryable vs non-retryable upstream error classification
- policy-driven retry/fallback behavior
- OpenAI adapter streaming support
- full OpenAI chat completions request/response compatibility
- richer request/response passthrough normalization
- structured request attempt history
- debuggable attempt-order logs and final route decision logs
- provider availability/config diagnostics at startup
- admin API or UI

## Recommended Next PRs

### PR 1: fallback policy and attempt logging

Goal: make existing provider fallback explicit, debuggable, and safe.

Suggested scope:

- classify upstream failures as retryable, fallback-eligible, or terminal
- preserve deterministic provider order within each provider type
- stop fallback on client/config/auth failures that should not be retried
- emit logs for requested model, resolved target, provider attempt index, outcome, latency, and final error category
- add table-driven tests for fallback eligibility

### PR 2: OpenAI compatibility and streaming

Goal: make the OpenAI-compatible surface usable by more clients.

Suggested scope:

- expand `internal/openaiwire` for common chat completion fields such as `temperature`, `max_tokens`, `tools`, `tool_choice`, `usage`, and `finish_reason`
- add OpenAI upstream streaming support or return a clearer unsupported-streaming error for providers that cannot stream
- normalize upstream response model IDs back to the client-facing prefixed model
- add OpenAI adapter unit tests with a mock `http.Client`
- add compatibility tests for common OpenAI-style client payloads

### PR 3: models, diagnostics, and observability

Goal: make the API easier to inspect and operate.

Suggested scope:

- improve `/v1/models` to reflect system catalog models more completely
- expose provider availability/config validation issues clearly at startup
- tighten error responses and add more table-driven tests
- add structured request attempt history
- decide whether request history is in-memory only for MVP or persisted later

## Small but important fixes already worth landing

- document the current config schema as implemented, including required provider `id`, `type`, and `name`
- decide how custom OpenAI-compatible base URLs should be configured, if needed
- keep bootstrap architecture simple while adding policy and observability incrementally
