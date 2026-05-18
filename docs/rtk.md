# RTK Request Compression

This document describes the RTK request-compression layer as implemented in `goroute`.

## Scope

RTK is a default-on, deterministic compression pass that runs immediately before upstream request marshalling.

Current behavior:

- enabled by default and configurable through runtime settings and `~/.goroute/config.json`
- applies only to large machine-generated `user` and `tool` text
- skips `system`, `assistant`, and `instructions`
- keeps the OpenAI-compatible wire contract intact
- records request-level RTK diagnostics in `rtk_records`

RTK is not AI summarization and does not rewrite prompts semantically.

## Integration Points

RTK currently runs at the provider adapter boundary:

- OpenAI upstream:
  - `(*Client).ChatCompletions(...)`
  - `(*Client).ChatCompletionsStream(...)`
  - `(*Client).Responses(...)`
  - `(*Client).ResponsesStream(...)`
- Codex upstream:
  - after chat-completions is translated into `ResponsesRequest`
  - before the translated payload is marshalled and sent upstream

This means:

- `translated_request_body` reflects the compressed payload
- existing redaction still applies before persistence
- runtime setting changes apply immediately because adapters read RTK state from request context

## Supported Fields

RTK currently considers only text-bearing request fields:

- `messages[].content` when it is a string
- `messages[].content[].text`
- `ResponsesRequest.InputText`
- `input[].content[].text` for responses-style message items
- `function_call_output.output`
- raw `/v1/responses` JSON `input` payloads are compressed while preserving unknown passthrough fields

RTK does not touch:

- `model`
- `tools`
- `tool_choice`
- JSON schema payloads
- image fields and image URLs
- `instructions`
- non-text structured fields

## Filters

Implemented filters:

- `dedup-log`
- `ls`
- `tree`
- `read-numbered`
- `search-list`
- `smart-truncate`

High-level behavior:

- specialized pattern first
- `dedup-log` for repetitive output
- `smart-truncate` only when still large and no specialized summary should own the output
- parser/detector failures fall back to the original input

## Configuration

Config file examples:

Default enabled:

```json
{
  "rtk": {
    "enabled": true
  }
}
```

Disabled:

```json
{
  "rtk": false
}
```

Admin settings API shape:

```json
{
  "rtk": {
    "enabled": true
  }
}
```

## Diagnostics

Each completed request may produce zero or one RTK diagnostics row in `rtk_records`.

Implemented fields:

- `id`
- `run_id`
- `request_id`
- `applied`
- `bytes_before`
- `bytes_after`
- `saved_bytes`
- `saved_percent`
- `filter_chain`
- `hit_count`
- `field_count`

`run_id` links RTK diagnostics back to `ai_request_runs.id`, while `request_id` keeps the stable request identifier aligned with flow and third-party log records.
