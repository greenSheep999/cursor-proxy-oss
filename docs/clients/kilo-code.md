# Kilo Code (VS Code)

Roo Cline / Kilo Code fork with more provider adapters. Same wire shape
as Cline (OpenAI Chat Completions).

## Install

Search "Kilo Code" in the VS Code marketplace.

## Configure

In Kilo Code settings:

- **API Provider**: `OpenAI Compatible` (or "OpenRouter" — either works
  against `cursor-proxy`).
- **Base URL**: `http://localhost:8317/v1`
- **API Key**: your `$SK`
- **Model ID**: `composer-2.5`, `claude-4.5-sonnet`, `gpt-5`, etc.

If Kilo Code sends `x-api-key` in OpenRouter mode instead of
`Authorization: Bearer`, that's fine — `cursor-proxy` accepts both.

## Notes

- Kilo Code sometimes attaches OpenRouter-style headers (`HTTP-Referer`,
  `X-Title`) so requests can be attributed on OpenRouter's dashboard.
  `cursor-proxy` ignores them silently — no error.
- Streaming, tool use, and Kilo Code's "multi-model" mode all work.
