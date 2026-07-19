# cursor-proxy

> Tracking **Cursor kernel `3.11`** &nbsp;·&nbsp; current image tag `cursor3.11-v0.3.1` &nbsp;·&nbsp;
> [release history](https://github.com/greenSheep999/cursor-proxy-oss/releases)

Run any modern coding-agent CLI against your own Cursor Pro subscription.

`cursor-proxy` is a small local service that speaks the major public LLM API
shapes on one port and forwards the actual generation to Cursor's backend
using your existing Cursor account. From `v0.3.0` it also embeds Cursor's
official `@cursor/sdk` for full **agent mode** (indexing, MCP servers, skills,
hooks) — see [Agent mode](#agent-mode).

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

One HTTP endpoint that speaks **four provider shapes** at once, plus
`cursor-proxy`-specific ops endpoints for capabilities, usage, and
agent orchestration.

### Wire mode (chat / completions / messages / gemini)

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

### Ops / observability (unauthenticated)

| Route | Purpose |
|---|---|
| `GET /v1/proxy-info` | Build info, active wire/agent modes, Cursor account snapshot |
| `GET /v1/capabilities` | Compile-time feature matrix (streaming / caching / tools / thinking / …) |
| `GET /v1/introspect/recent-tools` | Tools declared by clients in the last N seconds |
| `GET /v1/introspect/recent-mcp-servers` | Same window projected onto MCP servers |
| `GET /v1/usage` | Current Cursor account quota & usage snapshot |
| `GET /v1/usage/events` | Paginated per-request event log (token breakdown, model, latency) |

### Agent mode (optional, since `cursor3.11/v0.3.0`)

| Route | Purpose |
|---|---|
| `POST /v1/agents` | Create an agent (local or cloud runtime) |
| `GET /v1/agents`, `GET /v1/agents/{id}` | List / describe agents |
| `POST /v1/agents/{id}/runs` | Run a prompt (streaming or one-shot) |
| `DELETE /v1/agents/{id}` | Close an agent (idempotent) |

Powered by `@cursor/sdk` under a Node runner. See
[docs/agents.md](docs/agents.md) for setup.

### Authentication

Wire and ops routes accept four key sources: `Authorization: Bearer`,
`x-api-key`, `x-goog-api-key`, and `?key=<APIKEY>` — pick whichever
your client sends. Ops routes marked *unauthenticated* skip the key
gate so ops probes and sidecar supervisors can read them freely
(no secrets are exposed).

## Features

- **Multi-provider on one port** — OpenAI (chat/legacy/responses),
  Anthropic Messages, Gemini native, all served side-by-side.
- **True SSE streaming** for chat/completions and messages
  (per-token stream from Cursor's backend).
- **Multi-turn tool loop** — `tool_use` / `tool_result` history
  threads coherently, tool inputs delivered as valid JSON.
- **Prompt-cache accounting** — `cache_read_input_tokens` /
  `cache_creation_input_tokens` from Cursor's upstream, blended
  with an in-proxy LRU-TTL simcache for realistic counts.
- **MCP tools passthrough** — `mcp__server__tool` names flow to
  Cursor MCP unchanged, so Claude Code / aider / cline all
  "just work" with your MCP fleet.
- **Extended thinking** — Anthropic thinking blocks preserved for
  reasoning models (`*-thinking-*`).
- **Model aliasing & effort mapping** — canonical Anthropic model
  names auto-map to Cursor's tier form; `output_config.effort`
  and `thinking.budget_tokens` map onto `-low/-medium/-high/-xhigh/-max`
  suffixes.
- **Multi-key auth** — comma-separated `CURSOR_PROXY_API_KEYS`,
  every client uses its own key.
- **Upstream egress proxy** — `HTTPS_PROXY` / `HTTP_PROXY` unlock
  geo-gated model families (`claude-*` / `gpt-*` / `gemini-*`).
- **Ops surface** — `/v1/proxy-info` + `/v1/capabilities` +
  `/v1/introspect/*` + `/v1/usage*` for probes and dashboards.
- **Agent mode (`v0.3.0+`)** — full `@cursor/sdk` integration:
  codebase indexing, MCP server management, skills, hooks, cloud
  runtimes with auto-PR.
- **Multi-arch Docker image** — `linux/amd64` + `linux/arm64`,
  works on Intel/AMD servers, Apple Silicon, Raspberry Pi.
- **Version pinned to upstream kernel** — image tag is
  `cursor<kernel>-v<semver>` (e.g. `cursor3.11-v0.3.1`), so you
  always know which Cursor client version is being impersonated.

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

## Endpoint reference

- [All endpoints](docs/endpoints.md) — full request/response shapes,
  auth rules, and error cases for every route.
- [Observability endpoints](docs/observability.md) — `/v1/capabilities`,
  `/v1/introspect/*`, `/v1/usage*`, `/v1/proxy-info`.
- [Agent mode](docs/agents.md) — `/v1/agents/*` setup and usage.

## Agent mode

`cursor3.11/v0.3.0` adds an optional **agent mode** — a `/v1/agents/*`
HTTP surface backed by the official `@cursor/sdk` Node package. It runs
alongside wire mode (both share the same process, `-api-keys` gate,
and Cursor account) and unlocks capabilities the wire protocol can't
express: codebase indexing, MCP server management, skills, hooks, and
cloud runtimes with auto-PR.

Wire mode keeps working exactly as before if you skip agent mode.

Quick enable (Docker):

```bash
docker run -p 127.0.0.1:8317:8317 \
  -e CURSOR_PROXY_API_KEYS=sk-cp-... \
  -e CURSOR_API_KEY=crsr_... \
  ghcr.io/greensheep999/cursor-proxy:cursor3.11-v0.3.5
```

`CURSOR_API_KEY` is your Cursor dashboard-issued key (`crsr_...`),
distinct from the IDE `accessToken` used by wire mode. See
[docs/agents.md](docs/agents.md) for the full HTTP surface, cloud
runtimes, and troubleshooting.

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

## Version tracking

Image tags follow the upstream kernel line and semver:

```
ghcr.io/greensheep999/cursor-proxy:cursor3.11-v0.3.5<X.Y>-v<x.y.z>
```

- `<X.Y>` — Cursor kernel version being impersonated (e.g. `3.11`).
  Different kernel lines are wire-incompatible; pick the one that
  matches the Cursor IDE build your account was provisioned against.
- `<x.y.z>` — `cursor-proxy` semver.
- `latest` and `cursor<X.Y>-latest` — moving pointers (avoid in
  production; pin to an explicit `cursor<X.Y>-v<x.y.z>`).

Git tags in this repo mirror image tags 1-for-1 (`cursor3.11-v0.3.1`
etc.). They are refreshed automatically on every upstream release by
`.github/workflows/sync-from-upstream.yml`.

**Major milestones** (with kernel version):

| Release | Kernel | Headline |
|---|---|---|
| [`cursor3.11-v0.3.1`](https://github.com/greenSheep999/cursor-proxy-oss/releases/tag/cursor3.11-v0.3.1) | `3.11` | Agent mode via `@cursor/sdk` + `/v1/usage/events` |
| [`cursor3.11-v0.2.7`](https://github.com/greenSheep999/cursor-proxy-oss/releases/tag/cursor3.11-v0.2.7) | `3.11` | Observability endpoints + per-window token breakdown |
| [`cursor3.11-v0.2.3`](https://github.com/greenSheep999/cursor-proxy-oss/releases/tag/cursor3.11-v0.2.3) | `3.11` | Kernel 3.11 bump + Anthropic compat hardening |

Full history: [releases page](https://github.com/greenSheep999/cursor-proxy-oss/releases).

**Older kernel line (`cursor3.10`)** &nbsp;·&nbsp; The `cursor3.10-*`
image tags on GHCR track the Cursor 3.10.20 kernel line for accounts
provisioned against that older client build. They are maintenance-only
— agent mode is not back-ported, and new features land on `3.11` first.
If you're setting up a fresh install, use `cursor3.11-*`.
Latest 3.10 image: `ghcr.io/greensheep999/cursor-proxy:cursor3.11-v0.3.5`.

### Project history (pre-v0.2.3)

The project bootstrapped from a full protobuf reverse of Cursor's
kernel 3.10.20 wire protocol. In its first ~48 hours it went from a
single Anthropic-shaped endpoint to a multi-provider proxy with:

- OpenAI Chat / Legacy / Responses, Anthropic Messages + count_tokens,
  Gemini native
- Multi-turn conversation history + `tool_use` / `tool_result` threading
- MCP tools passthrough
- Prompt-cache counter passthrough + local LRU-TTL simcache
- `/v1/usage` account quota snapshot
- Multi-key API auth, `-token-file` account loader, upstream egress
  proxy for geo-gated model families
- Multi-arch Docker image published to GHCR

`v0.2.x` then bumped the kernel to Cursor 3.11 and hardened
Anthropic compatibility. `v0.3.x` added agent mode via the official
Cursor SDK.

## Notes

- **Non-affiliated with Cursor / Anysphere.** This project uses your own Cursor Pro account against Cursor's servers. You are responsible for compliance with Cursor's Terms of Service.
- **No account is provided.** You need an active Cursor Pro subscription.
- The source of the wire-protocol layer is not part of this repository. The compiled multi-arch image is published at [ghcr.io/greensheep999/cursor-proxy](https://ghcr.io/greensheep999/cursor-proxy).

## License

Apache 2.0. See [LICENSE](LICENSE).
