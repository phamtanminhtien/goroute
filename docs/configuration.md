# Configuration and Data Model

## Configuration

The user config file is expected at `~/.goroute/config.json`.
It should configure local runtime behavior and credentials only; it should not define drivers, model namespaces, or model catalogs.

The current schema has two top-level domains:

- server
- providers

Example `~/.goroute/config.json`:

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
      "name": "[EMAIL_ADDRESS]"
    },
    {
      "id": "openai-1",
      "type": "openai",
      "api_key": "${OPENAI_API_KEY}",
      "name": "[EMAIL_ADDRESS]"
    }
  ]
}
```

Current validation requires every provider to have `id`, `type`, and `name`.
`server.listen` defaults to `:2232` when omitted.
Provider credentials are validated lazily by the selected provider adapter during request execution.

Current provider credential behavior:

- `codex` uses `access_token`, falling back to `api_key` if present.
- `openai` uses `api_key`, falling back to `access_token` if present.
- `refresh_token` is represented in config but is not used for refresh yet.

## System Data

System data currently lives in `data/system-drivers.json`.

Current shape:

```json
{
  "driver_auth_types": ["oauth", "api_key"],
  "drivers": [
    {
      "id": "cx",
      "name": "Codex",
      "provider": "codex",
      "auth_type": "oauth",
      "default_model": "cx/gpt-5.4",
      "models": [
        {
          "id": "cx/gpt-5.4",
          "name": "GPT-5.4",
          "description": ""
        }
      ]
    },
    {
      "id": "openai",
      "name": "OpenAI",
      "provider": "openai",
      "auth_type": "api_key",
      "default_model": "openai/gpt-4.1",
      "models": [
        {
          "id": "openai/gpt-4.1",
          "name": "GPT-4.1",
          "description": ""
        }
      ]
    }
  ]
}
```

## Data Model

### Provider config

Implemented fields:

- id
- type
- name
- api_key
- access_token
- refresh_token

### System driver definition

Implemented fields:

- driver ID / client-facing prefix, such as `cx`
- display name
- provider, such as `codex` or `openai`
- auth type, such as `oauth` or `api_key`
- default model
- supported models
- optional metadata

### Resolved execution target

At request time, routing produces:

- client-facing prefix
- requested upstream model
- driver ID and name
- provider type

Provider selection and fallback currently happen in the provider registry after resolution.
Fallback attempt index and request-scoped timeout metadata are not yet represented in the resolved target.
