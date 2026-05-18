# Design Overview

## Problem Statement

OpenAI-compatible clients are easy to integrate, but real deployments usually need more than a direct 1:1 connection to a single upstream:

- model names differ across upstreams
- credentials and base URLs vary by upstream
- fallback is often needed for reliability
- clients should not need connection-specific configuration
- routing policy should live server-side, not in every client

`goroute` centralizes that logic in one HTTP service.

## Scope

### In scope

- OpenAI-compatible HTTP ingress
- model prefix resolution
- connection selection
- provider/connection fallback chains
- request/response passthrough with minimal normalization
- structured logging and debuggable routing behavior
- admin APIs and UI for local connection/settings operations
- persisted request diagnostics for analytics and debugging
- deterministic request compression for large machine-generated request text

### Out of scope for the first useful version

- full connection-specific feature parity
- aggressive request mutation or prompt rewriting
- dynamic policy engines before static routing works well
- large control-plane features unrelated to request routing

## Current Implemented Surface

Implemented public routes:

- `GET /healthz`
- `GET /v1/models`
- `POST /v1/chat/completions`
- `POST /v1/responses`

Implemented admin routes are protected by bearer-token auth:

- provider catalog, OAuth start, and provider test routes
- connection CRUD, OAuth completion, and Codex usage lookup routes
- runtime settings routes for LLM logging and RTK
- usage analytics summary, timeseries, provider breakdown, and recent-request routes
- live log streaming for the admin console

Built-in providers are:

- `cx` for Codex
- `openai` for the standard OpenAI upstream

## High-level Architecture

```text
Client
  |
  v
HTTP API layer
  |
  v
Request validation
  |
  v
Model prefix resolution
  |
  v
Provider/connection selection
  |
  +--> primary upstream
  |
  +--> fallback upstream(s)
  |
  v
Connection adapter
  |
  v
Upstream HTTP call
  |
  v
Response normalization
  |
  v
Client response
```

Core idea: keep the data path simple and explicit.

## HTTP Surface

Current OpenAI-compatible endpoints:

- `POST /v1/chat/completions`
- `POST /v1/responses`
- `GET /v1/models`

Possible later endpoints:

- embeddings
- additional OpenAI-compatible endpoints as needed

The intended API contract is:

- clients talk to `goroute` as if it were an OpenAI-style server
- `goroute` keeps request and response shapes as close as possible to that contract
- connection-specific quirks are isolated inside adapters

## Request Lifecycle

For a request like:

```http
POST /v1/chat/completions
Authorization: Bearer <token>
Content-Type: application/json
```

with body:

```json
{
  "model": "cx/gpt-5.4",
  "messages": [
    {"role": "user", "content": "hello"}
  ]
}
```

The intended flow is:

1. parse and validate the JSON body
2. extract requested model (`cx/gpt-5.4`)
3. split the model into provider prefix (`cx`) and upstream model (`gpt-5.4`)
4. resolve `cx` to the system-defined Codex provider
5. transform request only where required by the connection adapter
6. let the provider execute `gpt-5.4` through the matching user-configured connections
7. if eligible failure occurs, advance through the provider's fallback policy
8. normalize upstream response to OpenAI-compatible output
9. emit logs / metrics describing the route decision

## Connection Abstraction

Each connection adapter should encapsulate:

- base URL construction
- authentication headers
- connection-specific path mapping
- request shaping differences
- response normalization differences
- retryable vs non-retryable error classification

This keeps routing logic independent from wire-level connection quirks.

Providers and model namespaces are system-defined, not user-defined config.
A provider maps a client-facing model prefix to connection-specific execution behavior and owns a default model.
For example, when a client sends `cx/gpt-5.4`, the built-in `cx` provider runs `gpt-5.4` through the user-configured Codex connections.
Likewise, `openai/gpt-4.1` resolves to the built-in OpenAI provider.
If the client sends only `cx` or `openai`, the provider uses its system-defined `default_model`.

A connection definition will typically need:

- connection type
- display name / account identity
- type-specific credentials

## Fallback Behavior

Fallback should be deterministic and policy-driven.

Expected behavior:

- try the primary connection first
- classify upstream failure
- retry the same connection only when policy allows
- move to next fallback only for retryable / fallback-eligible failures
- return a final error with enough context for logs and operator diagnosis

Examples of fallback-eligible failures:

- upstream timeout
- transient 5xx
- rate limiting if policy permits alternate connection fallback
- temporary network failure

Examples of likely non-fallback failures:

- malformed client request
- unsupported parameters
- authentication failure at ingress
- static config error

This classification should be explicit in code, not inferred ad hoc.

## Error Handling Goals

The proxy should preserve debuggability without leaking secrets.

Desired properties:

- stable HTTP status codes
- consistent error envelope for client-facing failures
- request ID included in logs and optionally responses
- provider details visible in logs
- auth tokens redacted
- upstream raw bodies available only where safe / configured

A useful internal error model will likely need to distinguish:

- client input errors
- proxy validation/config errors
- upstream transport errors
- upstream provider errors
- retry exhaustion / fallback exhaustion

## Streaming Behavior

Streaming is implemented for the current chat-completions and responses paths across the OpenAI and Codex providers.

Important concerns:

- preserving SSE framing semantics
- handling upstream disconnects cleanly
- deciding fallback behavior once a partial stream has started
- surfacing cancellation correctly via request context
- avoiding buffered behavior that breaks client expectations

Fallback after a partial stream has started should remain conservative. Future policy work should make retry/fallback eligibility explicit for both sync and streaming paths.

## `GET /v1/models` Behavior

`GET /v1/models` returns the system provider catalog as OpenAI-style model records.

The model IDs are already client-facing prefixed IDs such as:

- `cx/gpt-5.4`
- `openai/gpt-4.1`

Each record includes metadata for provider ID, provider name, auth type, display name, description, default model, and whether the listed model is the provider default.

## Observability

The proxy emits structured logs containing:

- timestamp
- request ID
- endpoint
- requested model prefix
- requested upstream model
- selected provider
- selected upstream model
- fallback attempt index
- latency
- final status
- sanitized error category

Request diagnostics are persisted to SQLite for admin analytics:

- `ai_request_runs`
- optional `ai_request_flows`
- optional `third_party_request_logs`
- RTK diagnostics in `rtk_records`

Current useful analytics include:

- request count by prefix/provider
- latency by provider/model
- upstream error rate
- token and cost summaries where pricing metadata is available

Future observability work should make fallback rate, timeout count, and final route-decision logs more explicit.

## Security Notes

This service sits on credentials and prompt traffic, so basic hygiene matters from day one.

Requirements:

- never log API keys
- redact `Authorization` headers
- validate inbound body sizes
- use bounded server timeouts
- avoid panic-driven 500s with leaked internals
- be explicit about whether prompts/responses are logged
- keep admin API auth simple and mandatory when admin endpoints are added

If multi-tenant usage is ever considered later, auth and per-client isolation will need stronger design.
