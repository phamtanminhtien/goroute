# Design Overview

## Problem Statement

OpenAI-compatible clients are easy to integrate, but real deployments usually need more than a direct 1:1 connection to a single provider:

- model names differ across providers
- credentials and base URLs vary by upstream
- fallback is often needed for reliability
- clients should not need provider-specific configuration
- routing policy should live server-side, not in every client

`goroute` centralizes that logic in one HTTP service.

## Scope

### In scope

- OpenAI-compatible HTTP ingress
- model prefix resolution
- provider selection
- driver/provider fallback chains
- request/response passthrough with minimal normalization
- structured logging and debuggable routing behavior

### Out of scope for the first useful version

- full provider-specific feature parity
- aggressive request mutation or prompt rewriting
- dynamic policy engines before static routing works well
- large control-plane features unrelated to request routing

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
Driver/provider selection
  |
  +--> primary upstream
  |
  +--> fallback upstream(s)
  |
  v
Provider adapter
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

## Expected HTTP Surface

Initial target endpoints:

- `POST /v1/chat/completions`
- `GET /v1/models`

Possible later endpoints:

- `POST /v1/responses`
- embeddings
- streaming variants / SSE handling
- additional OpenAI-compatible endpoints as needed

The intended API contract is:

- clients talk to `goroute` as if it were an OpenAI-style server
- `goroute` keeps request and response shapes as close as possible to that contract
- provider-specific quirks are isolated inside adapters

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
3. split the model into driver prefix (`cx`) and upstream model (`gpt-5.4`)
4. resolve `cx` to the system-defined Codex driver
5. transform request only where required by provider adapter
6. let the driver execute `gpt-5.4` through the matching user-configured providers
7. if eligible failure occurs, advance through the driver's fallback policy
8. normalize upstream response to OpenAI-compatible output
9. emit logs / metrics describing the route decision

## Provider Abstraction

Each provider adapter should encapsulate:

- base URL construction
- authentication headers
- provider-specific path mapping
- request shaping differences
- response normalization differences
- retryable vs non-retryable error classification

This keeps routing logic independent from wire-level provider quirks.

Drivers and model namespaces are system-defined, not user-defined config.
A driver maps a client-facing model prefix to provider-specific execution behavior and owns a default model.
For example, when a client sends `cx/gpt-5.4`, the built-in `cx` driver runs `gpt-5.4` through the user-configured Codex providers.
Likewise, `opena/gpt-4.1` resolves to the built-in OpenAI driver.
If the client sends only `cx` or `opena`, the driver uses its system-defined `default_model`.

A provider definition will typically need:

- provider type
- display name / account identity
- type-specific credentials

## Fallback Behavior

Fallback should be deterministic and policy-driven.

Expected behavior:

- try the primary provider first
- classify upstream failure
- retry the same provider only when policy allows
- move to next fallback only for retryable / fallback-eligible failures
- return a final error with enough context for debugging

Examples of fallback-eligible failures:

- upstream timeout
- transient 5xx
- rate limiting if policy permits alternate provider fallback
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

## Streaming Considerations

Streaming support is likely to be one of the harder early features.

Important concerns:

- preserving SSE framing semantics
- handling upstream disconnects cleanly
- deciding fallback behavior once a partial stream has started
- surfacing cancellation correctly via request context
- avoiding buffered behavior that breaks client expectations

A reasonable strategy is to get non-streaming path correct first, then add streaming with targeted tests.

## `GET /v1/models` Behavior

This endpoint can be implemented in more than one way:

### Option A: expose system-defined model prefixes

Return client-facing prefixes only:

- `cx`
- `opena`

Pros:

- reflects what clients are expected to use
- stable and simple

Cons:

- hides upstream model inventory

### Option B: expose resolved upstream-backed virtual models

Return prefixed model IDs with metadata indicating underlying driver/provider behavior.

Pros:

- more transparent

Cons:

- less canonical if clients expect strict OpenAI shape

Recommendation: start with system-defined prefixes and keep output simple.

## Observability

At minimum, the proxy should emit structured logs containing:

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

Metrics can come later, but eventual useful metrics include:

- request count by prefix/provider
- latency by provider/model
- fallback rate
- upstream error rate
- timeout count

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
