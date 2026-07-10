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

- `gpt-5.6-sol-medium` — new-gen general (Sol variant)
- `gpt-5.6-terra-medium` — new-gen general (Terra variant)
- `gpt-5.6-luna-medium` — new-gen with light reasoning (Luna variant)
- `gpt-5.6-sol-high` / `-xhigh` / `-max` — same with more compute
- `gpt-5.5-medium` — previous-gen general
- `gpt-5-codex` — code-focused
- `composer-2.5` — Cursor's own model (no region gate, fastest)

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
- **Region gate**: If your Cursor account is served from a CN or HK IP,
  `gpt-*` model calls return HTTP 403 "Model not available in your
  region". Set `-upstream-proxy` (or `HTTPS_PROXY`) to a US/EU proxy on
  the cursor-proxy container. See [`../deployment/proxy.md`](../deployment/proxy.md).
  As a fallback without a proxy, `composer-2.5` and `grok-4.5-*` still
  work — they're Cursor-native and not gated.
