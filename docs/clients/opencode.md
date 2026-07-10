# opencode

TUI coding agent. Speaks OpenAI Chat Completions and, in some paths,
the legacy Completions endpoint.

## Install

```bash
brew install sst/tap/opencode
# or npm install -g opencode-ai
```

## Configure

`~/.config/opencode/config.json`:

```json
{
  "provider": {
    "openai-compat": {
      "options": {
        "baseURL": "http://localhost:8317/v1",
        "apiKey": "sk-cp-...replace-with-yours..."
      },
      "models": {
        "composer-2.5": {},
        "gpt-5": {},
        "claude-4.5-sonnet": {}
      }
    }
  }
}
```

Then:

```bash
opencode --provider openai-compat --model composer-2.5
```

## Notes

- Some opencode versions probe `/v1/completions` for legacy models —
  `cursor-proxy` implements that route and returns proper
  `object:"text_completion"` shape.
- If you see `HTTP 404` in opencode logs for `/v1/embeddings`, that's
  expected — Cursor's backend has no embeddings, and opencode falls
  back to skipping vector-search features gracefully.
