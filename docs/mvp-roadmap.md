# MVP Roadmap

This file tracks the current implementation state and the next work needed to make `goroute` more reliable as a daily routing proxy.

## Current State

Implemented now:

- OpenAI-compatible HTTP surface for:
  - `GET /v1/models`
  - `POST /v1/chat/completions`
  - `POST /v1/responses`
- Anthropic-compatible HTTP surface for `POST /v1/messages`
- `GET /healthz`
- bearer-token auth for admin APIs
- request ID and structured logging middleware
- system provider catalog from built-in provider registrations
- built-in providers:
  - `cx`
  - `openai`
- model prefix resolution such as `cx/gpt-5.4` and `openai/gpt-4.1`
- default model resolution when the client sends only a provider prefix
- connection execution interfaces near the chat-completion and responses use cases
- connection registry that selects connections by resolved provider
- deterministic fallback across multiple configured connections for the same provider
- OpenAI upstream adapter for sync and streaming chat completions
- OpenAI upstream adapter for sync and streaming responses
- Anthropic upstream adapter for sync and streaming Messages, Responses, and Chat Completions
- Codex adapter for sync and streaming chat completions
- Codex adapter for streaming responses and sync response reconstruction
- normalized upstream error wrapper mapped to gateway responses by the HTTP layer
- config defaulting before validation
- runtime settings API for LLM logging and RTK
- SQLite persistence for connections and request diagnostics
- admin API and React admin UI for providers, connections, settings, usage analytics, quota lookup, and console logs
- RTK deterministic request compression with persisted diagnostics
- tests for routing, HTTP contract shape, connection registry behavior, config validation, provider mapping, analytics APIs, admin APIs, frontend pages, and shared UI behavior

## Remaining MVP Work

- explicit retryable vs non-retryable upstream error classification
- policy-driven retry/fallback behavior
- richer attempt-order logs and final route-decision logs
- connection availability and config diagnostics at startup
- more complete OpenAI chat-completions and responses compatibility
- broader response normalization for model IDs, usage, finish reasons, and error envelopes
- clearer operator-facing diagnostics when analytics or optional logging stores are disabled
- body-size, timeout, and cancellation hardening for public and admin routes

## Recommended Next PRs

### PR 1: fallback policy and attempt diagnostics

Goal: make existing connection fallback explicit, debuggable, and safe.

Suggested scope:

- classify upstream failures as retryable, fallback-eligible, or terminal
- preserve deterministic connection order within each provider
- stop fallback on client/config/auth failures that should not be retried
- record connection attempt index, outcome, latency, and final error category
- add table-driven tests for fallback eligibility and route-decision logs

### PR 2: OpenAI compatibility hardening

Goal: make the OpenAI-compatible surface work with more clients and payloads.

Suggested scope:

- expand `internal/openaiwire` for common chat-completions and responses fields
- normalize upstream response model IDs back to client-facing prefixed model IDs where needed
- tighten usage and finish-reason mapping
- add compatibility tests for common OpenAI-style client payloads

### PR 3: diagnostics and operations

Goal: make the proxy easier to inspect and operate.

Suggested scope:

- expose connection availability/config validation issues clearly at startup and in admin APIs
- improve analytics empty states and disabled-logging messages
- add more request-log correlation between public responses, structured logs, and persisted records
- tighten body-size, timeout, and cancellation behavior with targeted tests

## Small But Important Fixes

- keep docs synchronized with `internal/transport/httpapi/server.go` as routes change
- decide how custom OpenAI-compatible base URLs should be configured, if needed
- keep the config file focused on local runtime settings while connections and request diagnostics remain in SQLite
