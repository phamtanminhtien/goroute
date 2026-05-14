# Configuration and Data Model

## Configuration Direction

The user config file is expected at `~/.goroute/config.json`.
It should configure local runtime behavior and credentials only; it should not define drivers, model namespaces, or model catalogs.

The exact schema is not finalized yet, but the likely config domains are:

- server
- providers

Illustrative `~/.goroute/config.json` example:

```json
{
  "server": {
    "listen": ":2232",
    "auth_token": "change-me"
  },
  "providers": [
    {
      "id": "1",
      "type": "codex",
      "access_token": "${ACCESS_TOKEN}",
      "refresh_token": "${REFRESH_TOKEN}",
      "name": "[EMAIL_ADDRESS]"
    },
    {
      "id": "2",
      "type": "openai",
      "api_key": "${OPENAI_API_KEY}",
      "name": "[EMAIL_ADDRESS]"
    }
  ]
}
```

This example is descriptive, not a committed spec.

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

Even before implementation, a few internal structures are implied.

### Provider config

Fields likely needed:

- name
- id
- type
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
