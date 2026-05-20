# Implementation Plan

This document tracks implementation direction from the current codebase. It is not a bootstrap blueprint; `goroute` already has the core HTTP data path, provider registry, admin UI, settings, analytics, and request diagnostics in place.

## Current Architecture

The request path follows this shape:

```text
HTTP ingress -> use case -> routing/domain policy -> provider connection registry -> upstream adapter
```

Important package boundaries:

- `cmd/goroute` owns process startup.
- `internal/app` wires config, storage, providers, use cases, HTTP handlers, and static web UI serving.
- `internal/transport/httpapi` owns public `/v1/*`, health, admin API handlers, auth, request IDs, logging middleware, and web UI fallback routing.
- `internal/usecase/*` owns application workflows for chat completions, responses, connections, list models, and analytics.
- `internal/domain/*` owns provider, routing, connection, and request-log records.
- `internal/adapter/provider/*` owns provider-specific execution and registration.
- `internal/adapter/upstream/*` owns reusable upstream protocol clients.
- `internal/openaiwire` owns OpenAI-compatible request/response structs.
- `internal/config` owns local runtime config and settings persistence.
- `web` owns the React admin UI and talks to the backend through `/admin/api/*`.

## Implemented Surface

Public API:

- `GET /healthz`
- `GET /v1/models`
- `POST /v1/chat/completions`
- `POST /v1/responses`

Admin API:

- provider catalog, OAuth URL generation, and provider model tests
- connection CRUD, OAuth completion, and Codex usage lookup
- runtime settings for LLM logging and RTK
- usage analytics summary, timeseries, provider breakdown, and recent request APIs
- live log streaming

Provider support:

- `cx` for Codex, with OAuth-oriented connection support and usage lookup
- `openai` for the standard OpenAI upstream, with API-key-oriented connection support
- `anthropic` for the standard Anthropic upstream, with API-key-oriented connection support
- sync and streaming execution for chat completions and responses
- deterministic fallback across configured connections for the resolved provider

Persistence:

- `~/.goroute/config.json` stores local server, logging, and RTK settings
- `~/.goroute/goroute.db` stores connections and request diagnostics
- `ai_request_runs` is always written
- `ai_request_flows`, `third_party_request_logs`, and `rtk_records` are controlled by settings and request behavior

## Next Implementation Work

### Fallback policy and diagnostics

- Make retryable, fallback-eligible, and terminal upstream failures explicit.
- Stop fallback on client input, config, and auth failures that should not be retried.
- Persist and expose attempt-order details for diagnostics.
- Add final route-decision logs that are easy to correlate with `request_id`.

### OpenAI wire compatibility

- Continue expanding common request and response fields for chat completions and responses.
- Normalize response model IDs consistently back to client-facing prefixed model IDs where needed.
- Add compatibility tests for common OpenAI-style clients and payload shapes.
- Keep provider-specific quirks inside adapters rather than handlers.

### Admin and analytics refinement

- Improve connection health/config diagnostics at startup and in the admin UI.
- Add clearer empty/error states for analytics when request logging is disabled or no rows match a filter.
- Keep Codex quota lookup live and separate from persisted request analytics.

### Operational hardening

- Tighten body-size, timeout, and cancellation behavior across public and admin routes.
- Add more table-driven tests around error envelopes and upstream failure classification.
- Keep secrets redacted in structured logs, persisted diagnostics, and UI responses.

## Testing Direction

Unit tests should continue to cover:

- model prefix resolution
- config defaulting, validation, and settings updates
- provider registration and connection validation
- fallback eligibility and attempt ordering
- OpenAI/Codex/Anthropic request and response mapping
- RTK compression selection and diagnostics

Integration tests should cover:

- handler to mock-upstream passthrough for sync and streaming paths
- admin auth and CRUD behavior
- analytics APIs over persisted request runs
- request logging persistence under different settings combinations
- stable error responses for invalid input and upstream failures
