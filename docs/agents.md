# Agent mode

**Introduced**: `cursor3.11/v0.3.0` &nbsp;·&nbsp; not back-ported to
the `cursor3.10` line.

Agent mode adds a `/v1/agents/*` HTTP surface backed by the official
[`@cursor/sdk`](https://www.npmjs.com/package/@cursor/sdk) Node
package. It runs alongside wire mode (`/v1/messages` etc.) — both
surfaces share the same process, the same `-api-keys` gate, and the
same Cursor account, but they use different authentication mechanisms
and unlock different capabilities.

If you don't need `/v1/agents/*`, ignore this document. Wire mode
keeps working exactly as it did in `v0.2.x`.

## Two modes, one process

|                  | Wire mode                               | Agent mode                                             |
|------------------|-----------------------------------------|--------------------------------------------------------|
| Endpoints        | `/v1/messages`, `/v1/chat/completions`… | `/v1/agents/*`                                         |
| Cursor auth      | IDE `accessToken` from `state.vscdb`    | Dashboard-issued `CURSOR_API_KEY` (`crsr_...`)         |
| Backend path     | Go executor → private protobuf          | `@cursor/sdk` in a Node child process                  |
| Node.js needed?  | No                                      | Yes (bundled in the Docker image)                      |
| Capabilities     | Chat, tool-loop, streaming              | + codebase indexing / MCP servers / skills / hooks     |
| Cloud runtime?   | No                                      | Yes (auto-PR against your repo)                        |

Both modes go through the `-api-keys` allowlist for client
authentication.

## Enabling agent mode

Three paths, cheapest first.

### 1. Docker (recommended)

The published multi-arch image ships the Node runner at
`/opt/cursor-node-runner/dist/index.js` and pre-sets the runner
path in the environment. Set `CURSOR_API_KEY` and you're done:

```bash
docker run -p 127.0.0.1:8317:8317 \
  -e CURSOR_PROXY_API_KEYS=sk-cp-... \
  -e CURSOR_API_KEY=crsr_... \
  ghcr.io/greensheep999/cursor-proxy:cursor3.11-v0.3.5
```

`docker-compose.yml` in this repo already exposes `CURSOR_API_KEY`
as an optional env var — set it in your `.env` and `docker compose
up -d`.

### 2. Combined tarball

Download the release tarball from the [releases page](https://github.com/greenSheep999/cursor-proxy-oss/releases/tag/cursor3.11-v0.3.1),
extract, and:

```bash
tar -xzf cursor-proxy-cursor3.11-linux-amd64.tar.gz
cd cursor-proxy-cursor3.11-linux-amd64

CURSOR_API_KEY=crsr_... ./cursor-proxy \
  -addr 127.0.0.1:8317 \
  -node-runner ./cursor-node-runner/dist/index.js
```

Requires **Node 22.13+** on `PATH` (`nvm install 22` or Homebrew's
`node@22`).

### 3. From source

Not shipped from this OSS repository — the wire-protocol layer
is compiled from a private core repo. If you have access, see the
core repo's build instructions.

## Getting a `CURSOR_API_KEY`

`CURSOR_API_KEY` is a dashboard-issued key distinct from the IDE
`accessToken` used by wire mode:

1. Sign in to Cursor at <https://cursor.com>.
2. Open Dashboard → **API Keys** → **Create new key**.
3. Copy the `crsr_...` token. This is your `CURSOR_API_KEY`.

You can revoke it from the same dashboard any time; the proxy will
surface `agent_mode.available: false` on next `/v1/proxy-info` and
existing agents will fail on their next run.

## Verifying it's up

`GET /v1/proxy-info` (unauthenticated) reports both modes:

```json
{
  "proto_version": "cursor3.11/v0.3.1",
  "wire_mode":  {"available": true, "account_email": "u@example.com"},
  "agent_mode": {
    "available":    true,
    "sdk_version":  "1.0.23",
    "node_version": "v22.13.0",
    "runtimes":     ["local", "cloud"],
    "active_agents": 0,
    "active_runs":   0
  }
}
```

If `agent_mode.available == false`, the startup log will explain why:

- `[proxy] agent mode: CURSOR_API_KEY is set but -node-runner is empty; agent mode disabled`
- `[proxy] agent mode: -node-runner set but no CURSOR_API_KEY; agent mode disabled`
- `[proxy] agent mode: node runner failed to start (…); agent mode disabled`

## HTTP surface

All endpoints require the `-api-keys` gate (same as `/v1/messages`).

### Create an agent

```bash
curl -X POST -H "x-api-key: $KEY" -H "Content-Type: application/json" \
  http://127.0.0.1:8317/v1/agents \
  -d '{
    "runtime": "local",
    "cwd":     "/path/to/repo",
    "model":   {"id": "composer-2.5"}
  }'
```

Returns `{"agentId": "agent-<uuid>", "createdAt": "..."}`.

For a **cloud** runtime (Cursor spins up its own sandbox, checks out
your repo, and optionally opens a PR when done):

```json
{
  "runtime": "cloud",
  "model":   {"id": "composer-2.5"},
  "repos":   [{"url": "https://github.com/your-org/your-repo"}],
  "auto_create_pr": true
}
```

### List / describe / close

```bash
curl -H "x-api-key: $KEY" http://127.0.0.1:8317/v1/agents
curl -H "x-api-key: $KEY" http://127.0.0.1:8317/v1/agents/agent-abc
curl -X DELETE -H "x-api-key: $KEY" http://127.0.0.1:8317/v1/agents/agent-abc
```

`DELETE` is idempotent: closing an already-gone agent returns
`{"ok": true, "already_gone": true}`.

### Run a prompt (non-streaming)

```bash
curl -X POST -H "x-api-key: $KEY" -H "Content-Type: application/json" \
  http://127.0.0.1:8317/v1/agents/agent-abc/runs \
  -d '{"prompt": "Summarize this repo", "stream": false}'
```

Response includes the final assistant message plus a `result`
object with token counts, tool calls made, and files touched
(local runtime) or PR URL (cloud runtime).

### Run a prompt (streaming)

```bash
curl -N -X POST -H "x-api-key: $KEY" -H "Content-Type: application/json" \
  http://127.0.0.1:8317/v1/agents/agent-abc/runs \
  -d '{"prompt": "Refactor auth.go", "stream": true}'
```

Emits SSE frames: `run.started`, incremental `run.progress` (tool
calls, thinking, partial output), and terminal `run.completed` /
`run.failed`.

## Limits & known behavior

- **Single-tenant** — one Cursor account per process. Multi-tenant
  wrapping should happen at a higher layer.
- **Cloud runtime PRs require repo write** on the `CURSOR_API_KEY`'s
  account — Cursor uses your dashboard-linked GitHub App
  authorization to open the PR.
- **Node runner memory grows with active agents** — each idle agent
  is cheap (~30 MB), a running one is ~150-400 MB. `DELETE` when
  done rather than leaving them lying around.
- **Skills / hooks / MCP servers** configured on the underlying
  Cursor account are honored automatically; there's no per-request
  override right now.

## Troubleshooting

| Symptom | Fix |
|---|---|
| `agent_mode.available: false` at startup | Set both `CURSOR_API_KEY` and (in non-Docker installs) `-node-runner`. |
| `401 unauthorized` on `/v1/agents` | Missing or wrong `CURSOR_PROXY_API_KEYS` entry — this is the *client* gate, separate from `CURSOR_API_KEY`. |
| `400` "invalid CURSOR_API_KEY" | Token was revoked or belongs to a suspended account — regenerate at cursor.com/dashboard/api-keys. |
| `run.failed` with `runtime error: no such repo` | Cloud runtime got a repo URL your `CURSOR_API_KEY`'s account can't reach — check the GitHub App is installed on that org. |
| Node runner keeps restarting | Check container/host has enough memory; the runner will restart-with-backoff on crash and log the reason. |
