# Configuration and Data Model

## Current Config Schema

The user config file is loaded from `~/.goroute/config.json`.
It configures local runtime behavior and credentials only; it does not define drivers, model namespaces, or model catalogs.

The schema currently implemented in `internal/config` is:

```json
{
  "server": {
    "listen": ":2232",
    "auth_token": "change-me"
  },
  "providers": [
    {
      "id": "codex-primary",
      "type": "codex",
      "access_token": "${ACCESS_TOKEN}",
      "refresh_token": "${REFRESH_TOKEN}",
      "name": "user@example.com"
    },
    {
      "id": "openai-primary",
      "type": "openai",
      "api_key": "${OPENAI_API_KEY}",
      "name": "user@example.com"
    }
  ]
}
```

### Validation rules implemented now

- `providers` must contain at least one entry.
- Every provider must include:
  - `id`
  - `type`
  - `name`
- `server.listen` defaults to `:2232` when omitted.
- Provider-specific credentials are not yet deeply validated at config-load time; adapters fail later if no usable credential is present.

### Provider fields implemented now

#### Common fields

- `id`: stable local identifier for the configured provider entry
- `type`: upstream provider family, currently expected to match built-in adapters such as `codex` or `openai`
- `name`: display name or account identity used in logs and errors

#### Credential fields currently recognized

- `api_key`: used by OpenAI-compatible execution, and accepted as a fallback credential by the Codex adapter
- `access_token`: used by the Codex adapter, and also accepted by the OpenAI-compatible adapter as a fallback bearer token
- `refresh_token`: currently stored for Codex-oriented setups, but not yet actively refreshed by runtime logic

## Custom OpenAI-Compatible Base URL Direction

The current implementation uses the built-in OpenAI base URL (`https://api.openai.com`) for `type: "openai"` providers.
There is no config field yet for overriding this per provider.

That is an intentional bootstrap constraint for now:

- keep the config contract small while real execution settles
- avoid introducing an underspecified field before fallback and observability are in place
- leave room for other OpenAI-compatible upstreams without prematurely hard-coding policy

The likely future direction is a per-provider optional field such as `base_url` on OpenAI-compatible providers, rather than a global setting.
That would preserve the existing provider-centric config shape and allow multiple OpenAI-compatible accounts or vendors side by side.

Until that lands, `type: "openai"` should be read as “the standard OpenAI upstream” rather than “any OpenAI-compatible endpoint.”

## System Data

System data is expected to live in a JSON file for now.

Illustrative system data:

```json
{
  "driver_auth_types": ["oauth", "api_key"],
  "drivers": [
    {
      "id": "cx",
      "name": "Codex",
      "provider": "codex",
      "auth_type": "oauth",
      "models": [
        {
          "id": "cx/gpt-5.4",
          "name": "GPT-5.4",
          "description": ""
        }
      ]
    },
    {
      "id": "opena",
      "name": "OpenAI",
      "provider": "openai",
      "auth_type": "api_key",
      "models": [
        {
          "id": "opena/gpt-4.1",
          "name": "GPT-4.1",
          "description": ""
        }
      ]
    }
  ]
}
```

## Data Model Considerations

The bootstrap code already implies a few stable internal structures.

### Provider config

Fields implemented now:

- `name`
- `id`
- `type`
- type-specific credential fields, such as `api_key`
- type-specific session fields, such as `access_token` and `refresh_token`

### System driver definition

Fields likely needed:

- driver ID / client-facing prefix, such as `cx`
- display name
- provider, such as `codex` or `openai`
- auth type, such as `oauth` or `api_key`
- default model
- supported models
- optional metadata

### Resolved execution target

At request time, routing should produce a resolved structure containing:

- client-facing prefix
- requested upstream model
- selected driver
- selected provider
- fallback index / attempt state
- request-scoped timeout context

This resolved object should be what the execution path logs.
