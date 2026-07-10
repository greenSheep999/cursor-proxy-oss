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
echo "SK=$SK"

# 3. Start
docker compose up -d

# 4. Try it
curl http://localhost:8317/v1/chat/completions \
  -H "Authorization: Bearer $SK" \
  -H "content-type: application/json" \
  -d '{"model":"composer-2.5","messages":[{"role":"user","content":"hi"}]}'
```

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

The exact list depends on your Cursor account plan and region. All Cursor-provided models pass through — a typical Pro account exposes:

`composer-2.5`, `composer-2.5-fast`, `claude-4.5-sonnet`, `claude-4.5-haiku`, `claude-opus-4.1`, `gpt-5`, `gpt-5-mini`, `gpt-5-codex`, `gemini-2.5-pro`, `gemini-2.5-flash`, `grok-code`, `cursor-small`.

Call `GET /v1/models` after the proxy starts to see exactly what your account can access.

## Notes

- **Non-affiliated with Cursor / Anysphere.** This project uses your own Cursor Pro account against Cursor's servers. You are responsible for compliance with Cursor's Terms of Service.
- **No account is provided.** You need an active Cursor Pro subscription.
- The source of the wire-protocol layer is not part of this repository. The compiled multi-arch image is published at [ghcr.io/greensheep999/cursor-proxy](https://ghcr.io/greensheep999/cursor-proxy).

## License

Apache 2.0. See [LICENSE](LICENSE).
