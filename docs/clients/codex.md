# codex CLI

OpenAI's `codex` CLI talks the Responses API. Point its base URL at
`cursor-proxy` and it forwards straight through.

## Install

```bash
npm install -g @openai/codex
```

## Configure

```bash
export OPENAI_BASE_URL=http://localhost:8317/v1
export OPENAI_API_KEY=$SK    # the CURSOR_PROXY_API_KEYS value
```

Or persist in `~/.codex/config`:

```toml
model = "gpt-5"
base_url = "http://localhost:8317/v1"
```

## Recommended models

- `gpt-5` — general
- `gpt-5-codex` — code-focused
- `composer-2.5` — Cursor's own model

## Try

```bash
codex "write a hello-world in Go"
```

Streaming events (`response.output_text.delta`, ...) render live in the
terminal.

## Notes

- `codex` sends `Authorization: Bearer`. If you deploy behind a reverse
  proxy that strips it, set `x-api-key` instead — `cursor-proxy` accepts
  both.
- If your account is region-locked out of `claude-*`, use `gpt-5` /
  `composer-2.5` / `grok-code` instead.
