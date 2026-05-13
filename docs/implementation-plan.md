# Implementation Plan

## Directory Architecture

The implementation should keep the request path easy to follow:

```text
HTTP ingress -> application use case -> routing/domain policy -> provider adapter -> upstream
```

Proposed Go layout:

```text
cmd/goroute/
  main.go                         # process entrypoint only: load config, build app, start server

internal/app/
  app.go                          # dependency wiring for the executable
  lifecycle.go                    # startup/shutdown coordination

internal/transport/httpapi/
  server.go                       # router/server bootstrap
  middleware.go                   # auth, request ID, logging middleware
  errors.go                       # HTTP error rendering
  chat_completions.go             # POST /v1/chat/completions handler
  models.go                       # GET /v1/models handler

internal/usecase/chatcompletion/
  execute.go                      # chat completion orchestration
  request.go                      # use-case input/output DTOs
  stream.go                       # later: streaming execution path

internal/usecase/listmodels/
  list.go                         # client-facing model listing

internal/domain/driver/
  driver.go                       # system-defined driver model
  catalog.go                      # driver catalog lookup

internal/domain/routing/
  resolve.go                      # model prefix -> driver/model resolution
  policy.go                       # fallback/retry policy
  target.go                       # resolved execution target

internal/domain/provider/
  provider.go                     # provider identity and capability contracts
  errors.go                       # retry/fallback error categories

internal/adapter/provider/openai/
  client.go                       # OpenAI-compatible upstream HTTP adapter
  mapper.go                       # request/response mapping
  errors.go                       # upstream error classification

internal/adapter/provider/codex/
  client.go                       # Codex-compatible upstream adapter, when added
  mapper.go
  errors.go

internal/adapter/systemdata/
  embedded.go                     # embedded system driver data
  json.go                         # JSON decoder/validator for system data

internal/config/
  load.go                         # user config loading
  validate.go                     # config validation
  types.go                        # user config structs

internal/openaiwire/
  chat.go                         # OpenAI-compatible request/response structs
  model.go                        # /v1/models wire structs
  error.go                        # OpenAI-compatible error envelope

internal/logging/
  logging.go                      # structured logging setup
  request.go                      # request-scoped logger helpers

internal/health/
  health.go                       # liveness/readiness endpoints

internal/testutil/
  upstream.go                     # mock upstream server helpers
  config.go                       # test config builders

data/
  system-drivers.json             # packaged system-defined drivers/models

web/
  package.json                    # React UI package
  vite.config.ts                  # Vite/dev-server config, if using Vite
  src/
    main.tsx                      # React entrypoint
    app/
      app.tsx                     # top-level app shell
      routes.tsx                  # route definitions
    features/
      providers/                  # provider config screens
      routing/                    # driver/model route visibility
      requests/                   # request logs / recent attempts
      settings/                   # local server settings
    shared/
      api/                        # typed client for goroute admin API
      ui/                         # reusable UI components
      lib/                        # small frontend-only helpers
    styles/
      globals.css
```

The exact file names can move as implementation reveals better package shapes, but the boundaries should stay stable.

### Layer ownership

- `cmd/goroute` owns process startup only. It should not contain routing, provider, or HTTP handler logic.
- `internal/app` wires dependencies together. It may know concrete implementations, but it should not implement business behavior.
- `internal/transport/httpapi` owns HTTP concerns: routing, auth, request IDs, body parsing, status codes, and response rendering.
- `internal/usecase/*` owns application workflows. A handler should call a use case and then render its result.
- `internal/domain/*` owns durable routing concepts: drivers, provider capabilities, resolved targets, fallback policy, and error categories.
- `internal/adapter/*` owns outside-world details: upstream HTTP calls, embedded/JSON system data, provider-specific request mapping, and provider-specific error normalization.
- `internal/openaiwire` owns compatibility structs for the OpenAI-shaped public API. Domain/usecase packages should avoid depending on raw HTTP handlers.
- `internal/config` owns user configuration only. System-defined drivers and model namespaces belong in `data/` plus `internal/adapter/systemdata`.
- `web` owns the optional React UI. It should talk to the backend through explicit HTTP APIs and should not duplicate backend routing policy.

### Dependency direction

Keep dependencies pointing inward:

```text
cmd
  -> internal/app
      -> internal/transport/httpapi
      -> internal/usecase/*
      -> internal/domain/*
      -> internal/adapter/*

transport/httpapi -> usecase -> domain
adapter/provider  -> domain + openaiwire
adapter/systemdata -> domain
config -> no domain policy
web -> backend HTTP API only
```

Practical rules:

- HTTP layer should not own routing policy
- provider adapters should not own config loading
- fallback logic should live in the chat completion use case and routing policy, not spread across handlers
- domain packages should not import provider adapter packages
- wire structs should be converted at the edge of the system, not leaked everywhere
- system driver data should not be mixed into the user config schema
- React state should reflect backend state; it should not become a second source of truth for providers, drivers, or route policy

### Package naming guidance

- Prefer feature/use-case packages over broad utility buckets.
- Avoid names like `service`, `manager`, or `common` unless the responsibility is genuinely narrow.
- Keep provider adapters grouped by upstream protocol/provider so adding a new provider is mostly additive.
- Keep tests near the behavior they verify; reserve `internal/testutil` for reusable server/config fixtures.
- Use kebab-case for React file names, such as `provider-list.tsx`, `route-timeline.tsx`, and `settings-form.tsx`.

## Optional React UI

If `goroute` grows a UI, treat it as a separate client application inside the same repository, not as another Go package.
The backend remains the source of truth; React only reads and updates state through admin-oriented HTTP endpoints.

Recommended first UI scope:

- provider list and credential status, with secrets redacted
- configured server settings
- system driver/model catalog
- recent request attempts and fallback decisions
- health/readiness status

Recommended backend additions:

```text
internal/transport/httpapi/admin/
  providers.go                    # GET/PUT provider config views
  drivers.go                      # GET system driver/model catalog
  requests.go                     # GET recent request logs or attempts
  settings.go                     # GET/PUT editable local settings
```

Keep the public OpenAI-compatible API separate from admin UI APIs:

- `/v1/*` remains the OpenAI-compatible client surface
- `/admin/api/*` serves UI/admin data
- `/admin/*` can serve the built React app in production

For local development, run React with its own dev server and proxy API calls to the Go server:

```text
React dev server: http://localhost:5173
Go API server:    http://localhost:2232
Proxy target:     /admin/api -> http://localhost:2232/admin/api
```

For distribution, there are two reasonable options:

- embed the built `web/dist` assets into the Go binary with `embed.FS`
- ship the UI as a separate static asset bundle beside the Go binary

Start with the embedded option if the goal is a simple local tool: one binary, one port, fewer moving parts.

## Interface Direction

Interfaces should usually live on the consumer side, close to the use case or domain policy that needs them.
For example, `internal/usecase/chatcompletion` can define the minimal upstream execution port it needs, while concrete providers in `internal/adapter/provider/*` implement that port.

A useful provider execution interface might look conceptually like this:

```go
type ChatCompletionProvider interface {
    ChatCompletions(ctx context.Context, req *ChatCompletionRequest, target ResolvedTarget) (*ChatCompletionResponse, error)
}
```

And resolution might produce something like:

```go
type ResolvedTarget struct {
    Prefix       string
    DriverName   string
    ProviderName string
    Model        string
    Attempt      int
}
```

These are examples only, but the implementation should prefer small explicit interfaces over large generic abstractions.

Good placement defaults:

- request/response wire structs: `internal/openaiwire`
- execution orchestration interfaces: `internal/usecase/chatcompletion`
- routing data structures: `internal/domain/routing`
- provider identity/capability types: `internal/domain/provider`
- concrete upstream HTTP clients: `internal/adapter/provider/<provider>`

## Testing Plan

The first meaningful test coverage should target routing correctness, not just HTTP happy paths.

### Unit tests

- prefix resolution
- fallback ordering
- retry eligibility classification
- config validation

### Integration tests

- handler -> mock upstream passthrough
- timeout behavior
- fallback on upstream failure
- normalization of upstream errors

### Compatibility checks

- verify a real OpenAI-style client can talk to the proxy
- verify `model` prefixing works without client-side changes

Table-driven tests are a good fit for prefix resolution and policy behavior.

## Suggested Implementation Order

### Phase 1

- config structs + validation
- basic HTTP server
- request ID + logging middleware
- `POST /v1/chat/completions`
- single OpenAI-compatible upstream adapter
- static prefix mapping

### Phase 2

- multiple providers
- fallback chain execution
- retry policy
- `GET /v1/models`
- improved error envelopes

### Phase 3

- streaming support
- health/readiness endpoints
- metrics
- broader test coverage

This sequence keeps the system usable early while limiting architecture churn.

## Example Deployment Model

A local client might be configured with:

- Base URL: `http://localhost:2232/v1`
- API key: proxy token
- Model: `cx/gpt-5.4`

`goroute` then resolves `cx/gpt-5.4` as:

- `cx` -> system driver named `Codex`
- `gpt-5.4` -> the model passed to user-configured providers with `"type": "codex"`

The client remains unchanged while routing policy evolves server-side.

## Repository Direction

This repository currently serves as a design anchor for the implementation.

The next useful milestone is not "support everything", but:

1. accept OpenAI-style chat completion requests
2. resolve a system-defined model prefix
3. call one upstream provider successfully
4. return a compatible response
5. produce logs that make routing behavior obvious

If that path is solid, the rest can be layered on without turning the codebase into bun rieu architecture.
