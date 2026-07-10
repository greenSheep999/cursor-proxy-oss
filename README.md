# cursor-proxy

Run any modern coding-agent CLI against your own Cursor Pro subscription.

`cursor-proxy` is a small local service that speaks the four public LLM API
shapes on one port and forwards the actual generation to Cursor's backend
using your existing Cursor account.

```
┌──────────────────────────┐    OpenAI / Anthropic / Gemini / Responses    ┌──────────────────┐
│  codex / Claude Code     │  ─────────────────────────────────────────▶   │                  │
│  aider / cline / opencode│                                                │  cursor-proxy    │
│  Kilo Code / Gemini CLI  │  ◀─── SSE / JSON ──                            │  (this image)    │
└──────────────────────────┘                                                └────────┬─────────┘
                                                                                     │
                                                                            your Cursor Pro
                                                                                     │
                                                                                     ▼
                                                                            Cursor's backend
```

## What you get

One HTTP endpoint that speaks **all four** major provider shapes at once:

| Route | Shape | Used by |
|---|---|---|
| `POST /v1/chat/completions` | OpenAI Chat Completions | aider, cline, opencode, Kilo Code, gptme, LiteLLM, any OpenAI SDK |
| `POST /v1/completions` | OpenAI legacy Completions | opencode, older SDKs |
| `POST /v1/responses` | OpenAI Responses API | `codex` CLI |
| `POST /v1/messages` | Anthropic Messages | Claude Code, Anthropic SDK |
| `POST /v1/messages/count_tokens` | Anthropic count-tokens (estimator) | Anthropic SDK |
| `POST /v1beta/models/{model}:generateContent` | Gemini generateContent | Gemini CLI, `google-generativeai` |
| `POST /v1beta/models/{model}:streamGenerateContent` | Gemini streaming | Gemini CLI |
| `GET  /v1/models`, `GET /v1/models/{id}` | OpenAI model list/detail | all |
| `GET  /v1beta/models` | Gemini model list | Gemini SDK |

All routes support four key sources: `Authorization: Bearer`, `x-api-key`,
`x-goog-api-key`, and `?key=<APIKEY>` — pick whichever your client sends.

## Quick start

```bash
git clone https://github.com/greenSheep999/cursor-proxy-oss
cd cursor-proxy-oss

# 1. Drop a Cursor auth file at ./auth/current.json
#    (see docs/deployment/auth-file.md for how to prepare one)

# 2. Generate an API key and put it in .env
SK=sk-cp-$(openssl rand -hex 16)
echo "CURSOR_PROXY_API_KEYS=$SK" > .env

# 3. (Optional but usually required)  Set an outbound proxy for the
#    Anthropic / OpenAI / Gemini model families. Cursor's backend
#    geo-gates those upstreams by request IP — from a CN/HK egress
#    every claude-* / gpt-* / gemini-* request returns HTTP 403
#    "Model not available in your region". Point HTTPS_PROXY at a
#    US or EU proxy (http:// or socks5://) to unlock them.
echo "HTTPS_PROXY=http://127.0.0.1:10808" >> .env   # your local system proxy
# echo "HTTPS_PROXY=socks5://user:pass@host:1080" >> .env  # or a remote SOCKS5

echo "SK=$SK"

# 4. Start
docker compose up -d

# 5. Try it
curl http://localhost:8317/v1/chat/completions \
  -H "Authorization: Bearer $SK" \
  -H "content-type: application/json" \
  -d '{"model":"claude-sonnet-5-medium","messages":[{"role":"user","content":"hi"}]}'
```

> If you skip step 3, `composer-2.5` / `grok-4.5-*` / `kimi-*` / `glm-*` still
> work directly — those are Cursor-native models with no geo-gate. Only the
> Anthropic / OpenAI / Gemini families need the outbound proxy.
> See [`docs/deployment/proxy.md`](docs/deployment/proxy.md) for details.

## Supported clients

Ready-to-paste config for each mainstream coding CLI:

- [Claude Code](docs/clients/claude-code.md)
- [OpenAI `codex`](docs/clients/codex.md)
- [Gemini CLI](docs/clients/gemini-cli.md)
- [aider](docs/clients/aider.md)
- [cline](docs/clients/cline.md)
- [Kilo Code](docs/clients/kilo-code.md)
- [opencode](docs/clients/opencode.md)

## Deployment

- [Docker Compose](docs/deployment/docker.md)
- [Kubernetes](docs/deployment/kubernetes.md)
- [Preparing an auth file](docs/deployment/auth-file.md)
- [Upstream proxy (unlocks Claude / GPT / Gemini)](docs/deployment/proxy.md)

## Go SDK + `cpctl` CLI

Two Go packages live in this repo alongside the docs:

- **`sdk/`** — a small typed HTTP client for the proxy's public API.
  Covers Chat Completions (streaming + non-streaming), Anthropic
  Messages, models list/detail, token estimator, and a `Probe`
  health check. Zero third-party runtime deps.

  ```go
  client := sdk.NewClient(sdk.Config{
      BaseURL: "http://localhost:8317",
      APIKey:  os.Getenv("CURSOR_PROXY_API_KEY"),
  })
  resp, _ := client.ChatCompletion(ctx, sdk.ChatRequest{
      Model:    "composer-2.5",
      Messages: []sdk.Message{{Role: "user", Content: "hi"}},
  })
  fmt.Println(resp.Choices[0].Message.Content)
  ```

- **`cmd/cpctl/`** — an operator CLI built on top of the SDK. Verbs:

  ```bash
  cpctl health                 # probe + latency + model count
  cpctl models -o json         # list models
  cpctl chat "explain generics" -s     # one-shot chat, streamed
  cpctl count "some text"      # heuristic Anthropic-style token count
  cpctl keygen                 # print a fresh sk-cp-* key
  ```

Full runnable programs are in [`examples/go/`](examples/go/).

Build locally:

```bash
go build ./cmd/cpctl
./cpctl health
```

Or download the multi-platform binaries from the latest CI artifact
(builds for `linux|darwin|windows` × `amd64|arm64`).

## What models can I use?

Every Cursor model your account can reach passes through. A current Pro
account with the upstream proxy configured exposes ~190 models across
these families:

| Family | Examples |
|---|---|
| **Cursor-native** (no geo-gate) | `composer-2.5`, `composer-2.5-fast` |
| **GPT-5.6** ("sol" / "terra" / "luna" variants, each with low/medium/high/xhigh/max effort tiers, `*-fast` variants) | `gpt-5.6-sol-medium`, `gpt-5.6-terra-high`, `gpt-5.6-luna-max-fast` |
| **GPT-5.5** | `gpt-5.5-low`, `gpt-5.5-medium`, `gpt-5.5-high`, `gpt-5.5-max` (+ `-fast`) |
| **Claude 5 line** (sonnet-5, fable-5; `low/medium/high/xhigh/max` + `-thinking` for extended reasoning) | `claude-sonnet-5-medium`, `claude-fable-5-thinking-high`, `claude-sonnet-5-xhigh` |
| **Claude 4.x line** (opus-4-8, opus-4-7, 4.6-sonnet, 4.5-sonnet, 4.5-haiku, opus-4.1) | `claude-opus-4-8-medium`, `claude-4.6-sonnet-medium`, `claude-4.5-sonnet` |
| **Gemini 3.x** | `gemini-3.1-pro`, `gemini-3-flash`, `gemini-3.5-flash`, `gemini-2.5-flash` |
| **Grok 4.5** | `grok-4.5-medium`, `grok-4.5-high`, `grok-4.5-xhigh` (+ `-fast`) |
| **Others** | `kimi-k2.7-code`, `glm-5.2-high`, `glm-5.2-max` |

Effort tiers (`low` → `medium` → `high` → `xhigh` → `max`) trade latency
for output quality. `-fast` variants stream noticeably quicker at the
cost of a small quality drop. `-thinking` variants inject a chain-of-
thought pass before the final answer — the token counts include
`reasoning_tokens`.

Some models require **Max Mode** on your account (opus-4.1, some `-max`
variants). Those return HTTP 400 with `Max Mode Required` when your tier
is too low — no other endpoints are affected.

Call `GET /v1/models` after the proxy starts to see exactly what your
own account can reach:

```bash
curl http://localhost:8317/v1/models -H "Authorization: Bearer $SK" \
  | python3 -c "import sys,json; [print(m['id']) for m in json.load(sys.stdin)['data']]"
```

## Notes

- **Non-affiliated with Cursor / Anysphere.** This project uses your own Cursor Pro account against Cursor's servers. You are responsible for compliance with Cursor's Terms of Service.
- **No account is provided.** You need an active Cursor Pro subscription.
- The source of the wire-protocol layer is not part of this repository. The compiled multi-arch image is published at [ghcr.io/greensheep999/cursor-proxy](https://ghcr.io/greensheep999/cursor-proxy).

## License

Apache 2.0. See [LICENSE](LICENSE).
