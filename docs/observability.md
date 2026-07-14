# Observability endpoints

**Introduced**: `cursor3.11/v0.2.6` (with additions in `v0.2.7`
and `v0.3.1`).

`cursor-proxy` exposes several read-only endpoints for downstream
consumers (dashboards, sidecar supervisors, health probes) to see
what the proxy supports and what recently flowed through it.

All endpoints listed here **bypass `-api-keys` authentication** —
there are no secrets in the responses, and unprivileged supervisors
need to probe them before an API key has been wired.

## `GET /v1/proxy-info`

**Since** `cursor3.11/v0.2.1`.

Build info and runtime facts. Poll once at startup; the only fields
that change during a process lifetime are `active_agents` /
`active_runs`.

```json
{
  "proto_version":         "cursor3.11/v0.3.1",
  "cursor_line":           "3.11",
  "impersonated_version":  "3.11.19",
  "commit":                "abc1234",
  "wire_mode": {
    "available":      true,
    "account_email":  "u@example.com"
  },
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

`proto_version` mirrors the git tag the binary was built from
(e.g. `cursor3.11/v0.3.1`) or `main-<sha>` for CI builds off `main`.

## `GET /v1/capabilities`

**Since** `cursor3.11/v0.2.6`.

Static description of what protocol features this build supports.
The values here are compile-time facts — they change when the
codebase changes, never per-request. Fetch once at sidecar spawn.

```json
{
  "streaming": true,
  "tool_use_json_input": true,
  "multi_turn_tool_loop": true,
  "thinking": true,
  "prompt_caching": {
    "read_tokens_reported":   true,
    "write_tokens_reported":  true,
    "local_simulator":        true,
    "cache_control_honored":  false
  },
  "server_tools": false,
  "mcp_tools": true,
  "effort_mapping": true,
  "anthropic_model_aliases": true,
  "http_version_options": ["auto", "http1.1", "http1.0"]
}
```

Field meanings:

- **streaming** — `/v1/messages` and `/v1/chat/completions` support SSE.
- **tool_use_json_input** — `tool_use.input` is delivered as valid JSON.
  False before `v0.2.3` (protobuf bytes leaked); permanently true now.
- **multi_turn_tool_loop** — assistant `tool_use` + user `tool_result`
  history threads coherently. True since `v0.2.3`.
- **thinking** — Extended Thinking blocks emitted for reasoning models.
- **prompt_caching.read_tokens_reported** —
  `usage.cache_read_input_tokens` populated on responses.
- **prompt_caching.write_tokens_reported** —
  `usage.cache_creation_input_tokens` populated.
- **prompt_caching.local_simulator** — the in-proxy simcache is on
  (matches `-simulate-cache` at boot). If true, cache_read counters
  blend real-upstream with local-estimate.
- **prompt_caching.cache_control_honored** — request-side
  `cache_control` markers are forwarded upstream. False today —
  Cursor's caching is server-side opaque, honoring the marker would
  be misleading.
- **server_tools** — Anthropic server-side tools (`web_search_20250305`
  etc.) accepted. Always false; proxy returns `400` to fail-fast.
- **mcp_tools** — `mcp__server__tool` names flow through unchanged.
- **effort_mapping** — `output_config.effort` and
  `thinking.budget_tokens` map onto Cursor's tier suffixes
  (`-low` / `-medium` / `-high`).
- **anthropic_model_aliases** — canonical bare names like
  `claude-sonnet-4-5-20250929` get rewritten to Cursor's tier form.
- **http_version_options** — values the operator can pass to
  `-http-version` / `CURSOR_PROXY_HTTP_VERSION`.

## `GET /v1/introspect/recent-tools?since=<duration>`

**Since** `cursor3.11/v0.2.6`.

Aggregated view of tools that the client(s) declared in recent
requests. Backed by an in-memory ring buffer (~4096 observations,
roughly 5 minutes at 10 tools/request × 1 request/second). Not
persisted across restarts.

Query parameters:

- **since** — window size. Accepts Go duration syntax (`60s`,
  `5m`, `1h`) or a bare integer of seconds. Default `60s`; garbage
  values fall back to the default rather than erroring.

Response:

```json
{
  "since_seconds": 60,
  "sample_size":   9,
  "unique_tools": [
    {"name": "Bash",                             "requests": 3, "kind": "custom"},
    {"name": "mcp__filesystem__read_file",       "requests": 3, "kind": "mcp", "server": "filesystem"},
    {"name": "mcp__github__create_issue",        "requests": 1, "kind": "mcp", "server": "github"}
  ],
  "oldest_seconds": 59.9
}
```

- **sample_size** — total tool observations in the window (a
  request declaring 3 tools contributes 3). Compare against
  `sample_size` on `/v1/introspect/recent-mcp-servers` to derive
  the MCP-vs-custom ratio.
- **unique_tools** — deduplicated by name, sorted by `requests`
  descending. `kind` is either `"custom"` or `"mcp"`; `server` is
  populated only when kind is mcp.
- **oldest_seconds** — age of the oldest observation returned.
  When smaller than the requested window, you're seeing the whole
  ring (older data has aged out or the proxy hasn't been running
  long enough).

Recording rules:

- Server-side Anthropic tools (`web_search_20250305` etc.) are
  **not** recorded — they get 400-rejected upstream, and recording
  them would fake activity that never actually happened.
- Empty tool arrays and blank names are ignored.
- MCP name detection matches both `mcp__server__tool` (canonical
  Claude Code) and the looser `mcp_server_tool` form some aider
  builds emit.

## `GET /v1/introspect/recent-mcp-servers?since=<duration>`

**Since** `cursor3.11/v0.2.6`.

Same ring buffer, projected onto the MCP server dimension. Useful
for dashboards showing "your Claude Code is using these MCP
servers" without reaching into the client's own settings.

Response:

```json
{
  "since_seconds": 60,
  "sample_size":   9,
  "servers": [
    {
      "server":     "filesystem",
      "requests":   3,
      "tool_names": ["mcp__filesystem__read_file"]
    },
    {
      "server":     "github",
      "requests":   2,
      "tool_names": ["mcp__github__create_issue", "mcp__github__search_repos"]
    }
  ]
}
```

- **sample_size** — same window total as `/recent-tools` (all
  observations, including non-MCP), so downstream can compute
  "9 total observations, 5 of them MCP → 55% MCP traffic".
- **servers** — sorted by `requests` descending; `tool_names` per
  server is deduplicated and alphabetically sorted.

## `GET /v1/usage/events?window=<24h|7d|30d>&limit=<n>&cursor=<opaque>`

**Since** `cursor3.11/v0.3.1`.

Paginated per-request event log. One entry per wire request, with
timestamp, model, token breakdown, latency, and status.

Query parameters:

- **window** — `24h` (default), `7d`, or `30d`.
- **limit** — page size, default `100`, cap `1000`.
- **cursor** — opaque continuation token from a prior page's
  `next_cursor`.

Response:

```json
{
  "window": "24h",
  "events": [
    {
      "ts":                          "2026-07-14T12:00:00Z",
      "model":                       "claude-sonnet-5-medium",
      "input_tokens":                1234,
      "output_tokens":               567,
      "cache_read_input_tokens":     800,
      "cache_creation_input_tokens": 100,
      "latency_ms":                  4321,
      "status":                      "ok"
    }
  ],
  "next_cursor": null
}
```

`status` is one of `ok`, `client_error`, `upstream_error`, `timeout`.
`next_cursor` is `null` on the last page.

Per-window rollups (previously `/v1/usage` and dashboard summaries)
break tokens down as `input / output / cache_read / cache_write` ×
`24h / 7d / 30d`. See [`v0.2.7` release
notes](https://github.com/greenSheep999/cursor-proxy-oss/releases/tag/cursor3.11-v0.2.7).

## What these endpoints are NOT

- **Not configuration.** They observe what the client did; they
  don't let you change tool policy or add MCP servers. Managing
  MCP / hooks / skills is agent-mode territory (`/v1/agents/*`).
- **Not per-user.** `cursor-proxy` is single-tenant — one process,
  one Cursor account, one shared ring.
- **Not audit-grade.** The ring buffer is best-effort in-memory
  storage; a proxy restart or high-QPS traffic can age data out
  before you query. Use it for UI hints and diagnostics, not
  for compliance.
- **Not secret-bearing.** No prompts, message bodies, tool inputs,
  or account identifiers appear in any response. Only tool NAMES
  the client declared.

## Recommended poll cadence

| Endpoint | Cadence |
|---|---|
| `GET /v1/proxy-info` | Once at spawn; re-poll if you care about `active_agents/active_runs`. |
| `GET /v1/capabilities` | Once at spawn, cache for process lifetime. |
| `GET /v1/introspect/*` | Every 15–30s while a dashboard is open. |
| `GET /v1/usage` | On demand (backed by a live Cursor account call). |
| `GET /v1/usage/events` | On demand + pagination as user scrolls. |
