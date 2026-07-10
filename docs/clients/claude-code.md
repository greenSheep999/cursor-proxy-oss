# Claude Code

Anthropic's official terminal client. Speaks `/v1/messages`.

## Install

```bash
npm install -g @anthropic-ai/claude-code
```

## Configure

```bash
export ANTHROPIC_API_URL=http://localhost:8317
export ANTHROPIC_AUTH_TOKEN=$SK
```

The base URL is **without `/v1`** — the Anthropic SDK appends `/v1/messages`
itself.

## Recommended models

- `claude-4.5-sonnet` — best default
- `claude-4.5-haiku` — fast, cheap
- `claude-opus-4.1` — heaviest, most careful

## Try

```bash
claude "explain the differences between mutex and channel in Go"
```

## Notes

- Claude Code sends `x-api-key: $ANTHROPIC_AUTH_TOKEN`. `cursor-proxy`
  accepts that as a first-class auth channel — no need to shim
  `Authorization: Bearer`.
- The `count_tokens` calls Claude Code sometimes makes are answered by
  `/v1/messages/count_tokens` (a heuristic estimator — not Anthropic's
  real tokenizer). The returned number is approximate but shape-correct.
- If your Cursor account is CN-region and `claude-*` is blocked, this
  client won't work regardless of the proxy. Switch to a
  non-region-locked account or use `codex` against `gpt-5`.
