# aider

Popular file-editing coding CLI. Speaks OpenAI Chat Completions.

## Install

```bash
python -m pip install aider-chat
```

## Configure

Either export env vars or put in `~/.aider.conf.yml`:

```bash
export OPENAI_API_BASE=http://localhost:8317/v1
export OPENAI_API_KEY=$SK
```

## Recommended models

```bash
aider --model composer-2.5                          # Cursor-native, always works
aider --model claude-sonnet-5-medium                # Claude 5 line
aider --model claude-4.6-sonnet-medium              # Claude 4.6
aider --model gpt-5.6-sol-medium                    # GPT-5.6 new-gen
aider --model gemini-3.1-pro                        # Gemini 3
```

Non-Cursor models need the upstream proxy configured on cursor-proxy;
see [`../deployment/proxy.md`](../deployment/proxy.md).

## Anthropic mode

aider supports Anthropic-shape too — pointing it at `/v1/messages` also works:

```bash
export ANTHROPIC_API_BASE=http://localhost:8317
export ANTHROPIC_API_KEY=$SK
aider --model claude-4.5-sonnet
```

## Notes

- aider's optional `--embed` mode requests `/v1/embeddings`. `cursor-proxy`
  does not implement embeddings (Cursor's backend does not expose them).
  aider degrades to its local embedder when the endpoint returns 404 —
  no config change needed.
- Streaming, tool calls, and diff-repair prompts all pass through.
