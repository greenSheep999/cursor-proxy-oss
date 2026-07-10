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

Newest first — pick the effort tier that matches your latency budget:

- **Claude 5 line** (best for planning, refactor, hard code):
  - `claude-sonnet-5-medium` — best default
  - `claude-sonnet-5-high` — deeper reasoning
  - `claude-sonnet-5-thinking-high` — explicit chain-of-thought
  - `claude-fable-5-medium` — Fable variant
- **Claude 4.x line** (still solid, less compute):
  - `claude-opus-4-8-medium` — heavy, most careful
  - `claude-4.6-sonnet-medium`
  - `claude-4.5-sonnet` / `claude-4.5-haiku` — fastest of the Claude family
- Higher effort tiers (`-high`, `-xhigh`, `-max`) and `-fast` variants of
  each are also available — see `curl /v1/models` for the full list.

Some heavy variants (`claude-opus-4.1`, some `-max`) require Max Mode on
your Cursor account and return HTTP 400 `Max Mode Required` if your
tier is too low.

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
- **Region gate**: from a CN/HK egress, `claude-*` returns
  `HTTP 403 Model not available in your region`. Set `-upstream-proxy`
  (or `HTTPS_PROXY`) on the cursor-proxy container to a US/EU proxy.
  See [`../deployment/proxy.md`](../deployment/proxy.md).
