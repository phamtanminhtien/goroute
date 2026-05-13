# MVP Roadmap

This file turns the current bootstrap state into a short, execution-focused roadmap.

## Current State

Implemented now:

- OpenAI-compatible HTTP surface for:
  - `GET /v1/models`
  - `POST /v1/chat/completions`
- bearer-token auth middleware
- request ID and request logging middleware
- system driver catalog loading from `data/system-drivers.json`
- model prefix resolution such as `cx/gpt-5.4`
- basic tests for routing and HTTP contract shape

Not implemented yet:

- real upstream provider execution
- provider selection from configured providers
- fallback/retry behavior
- request/response passthrough normalization
- structured request attempt history
- admin API or UI

## Recommended Next PRs

### PR 1: real provider execution path

Goal: replace bootstrap chat response with a real upstream call.

Suggested scope:

- define a provider execution interface near `internal/usecase/chatcompletion`
- add first OpenAI-compatible upstream adapter
- map request/response to upstream with minimal transformation
- wire configured providers into app startup
- return normalized upstream errors

### PR 2: provider selection and fallback

Goal: support more than one configured provider per type and fail gracefully.

Suggested scope:

- select provider candidates by driver/provider type
- classify retryable vs non-retryable upstream errors
- attempt fallback according to a simple deterministic policy
- emit debuggable logs for attempt order and final resolution

### PR 3: models and observability improvements

Goal: make the API easier to inspect and operate.

Suggested scope:

- improve `/v1/models` to reflect system catalog models more completely
- expose provider availability/config validation issues clearly at startup
- tighten error responses and add more table-driven tests

## Small but important fixes already worth landing

- config defaulting should happen before validation and should persist in the loaded config
- keep bootstrap architecture simple while adding real execution incrementally
