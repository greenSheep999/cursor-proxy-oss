# Endpoints

All routes accept any of these key sources (checked in order, constant-time):

1. `Authorization: Bearer <key>` — OpenAI-style
2. `x-api-key: <key>` — Anthropic-style
3. `x-goog-api-key: <key>` — Gemini-style
4. `?key=<key>` — Gemini SDK query fallback

`<key>` must be one of the keys listed in `CURSOR_PROXY_API_KEYS`. A wrong key
on any channel returns `401`. A missing key returns `401`. A valid key on one
channel plus a wrong one on another still returns `401` — this is intentional
to prevent silent downgrade.

## `GET /v1/models` — OpenAI model list

```bash
curl http://localhost:8317/v1/models -H "Authorization: Bearer $SK"
```

Response:

```json
{"object":"list","data":[
  {"id":"composer-2.5","object":"model","owned_by":"cursor","created":1783653322},
  {"id":"claude-4.5-sonnet","object":"model","owned_by":"cursor",...},
  ...
]}
```

## `GET /v1/models/{id}` — Single-model detail

Returns the same shape for one model, or `404` with `error.code = "model_not_found"`.

## `POST /v1/chat/completions` — OpenAI Chat Completions

Standard OpenAI Chat Completions shape. Both streaming (`"stream": true`) and non-streaming are supported. `tools:[{type:"function", function:{name, description, parameters}}]` are forwarded.

```bash
curl http://localhost:8317/v1/chat/completions \
  -H "Authorization: Bearer $SK" -H "content-type: application/json" \
  -d '{"model":"claude-sonnet-5-medium","messages":[{"role":"user","content":"say hi"}]}'
```

Any model your `GET /v1/models` returns can be named here. Common
picks: `composer-2.5`, `claude-sonnet-5-medium`, `claude-fable-5-medium`,
`gpt-5.6-sol-medium`, `gemini-3.1-pro`, `grok-4.5-medium`. Non-Cursor-
native families need the upstream proxy configured — see
[`deployment/proxy.md`](deployment/proxy.md).

## `POST /v1/responses` — OpenAI Responses API

The shape used by the OpenAI `codex` CLI. Non-streaming returns a single JSON body with `output:[{type:"message", content:[{type:"output_text", text}]}]`. Streaming emits the full event sequence:

```
response.created
response.in_progress
response.output_item.added
response.content_part.added
response.output_text.delta   (repeated)
response.output_text.done
response.content_part.done
response.output_item.done
response.completed
```

Tool calls interleave as `output_item.added` (type `function_call`) with `response.function_call_arguments.delta` frames and their own `.done`.

## `POST /v1/completions` — OpenAI legacy Completions

Accepts `{model, prompt: string | []string, stream?}`. Response object type is `"text_completion"`. Prompt strings are wrapped in a single user message internally.

## `POST /v1/messages` — Anthropic Messages

Standard Anthropic Messages API. Supports both streaming and non-streaming.

```bash
curl http://localhost:8317/v1/messages \
  -H "x-api-key: $SK" -H "anthropic-version: 2023-06-01" -H "content-type: application/json" \
  -d '{"model":"claude-4.5-sonnet","max_tokens":100,"messages":[{"role":"user","content":"hi"}]}'
```

## `POST /v1/messages/count_tokens` — Token counter (heuristic)

```bash
curl http://localhost:8317/v1/messages/count_tokens \
  -H "x-api-key: $SK" -H "content-type: application/json" \
  -d '{"model":"claude-4.5-sonnet","messages":[{"role":"user","content":"hello world"}]}'
# {"input_tokens": 3}
```

Note: this is an estimator (approximately `runes / 3.5`, floor 1) because Cursor's backend does not expose a real tokenizer. Anthropic SDK contracts are satisfied; exact numbers may differ from Anthropic's own count.

## `GET /v1beta/models` — Gemini SDK model list

```bash
curl http://localhost:8317/v1beta/models -H "x-goog-api-key: $SK"
```

Response shape: `{"models":[{"name":"models/composer-2.5","baseModelId":"composer-2.5","displayName":"composer-2.5","supportedGenerationMethods":["generateContent","streamGenerateContent"]},...]}`.

## `POST /v1beta/models/{model}:generateContent` — Gemini non-streaming

```bash
curl "http://localhost:8317/v1beta/models/composer-2.5:generateContent" \
  -H "x-goog-api-key: $SK" -H "content-type: application/json" \
  -d '{"contents":[{"role":"user","parts":[{"text":"say hi"}]}]}'
```

Response includes `candidates[].content.parts[].text`, `usageMetadata`, and `modelVersion`. Tool calls come back as `functionCall` parts.

## `POST /v1beta/models/{model}:streamGenerateContent` — Gemini streaming

Same request body. Response is SSE-style: `data: {json}\n\n` frames, terminated by a chunk with `finishReason:"STOP"` and full `usageMetadata`.

## `GET /v1/usage` — Account usage snapshot

Returns the account's current usage/quota status straight from Cursor's dashboard API. Handy for verifying auth is wired up.

## `GET /v1/usage/events` — Paginated per-request event log

**Since** `cursor3.11/v0.3.1`.

Every wire request produces an event: timestamp, model, token
breakdown (input / output / cache_read / cache_creation), latency,
and status. Useful for building dashboards without scraping logs.

```bash
curl "http://localhost:8317/v1/usage/events?window=24h&limit=100" \
  -H "Authorization: Bearer $SK"
```

Query parameters:

- `window` — one of `24h`, `7d`, `30d`. Default `24h`.
- `limit` — page size, default `100`, cap `1000`.
- `cursor` — opaque continuation token from a prior page's `next_cursor`.

Response:

```json
{
  "events": [
    {
      "ts": "2026-07-14T12:00:00Z",
      "model": "claude-sonnet-5-medium",
      "input_tokens": 1234,
      "output_tokens": 567,
      "cache_read_input_tokens": 800,
      "cache_creation_input_tokens": 100,
      "latency_ms": 4321,
      "status": "ok"
    }
  ],
  "next_cursor": null
}
```

`next_cursor` is `null` on the last page. `status` is one of
`ok`, `client_error`, `upstream_error`, `timeout`.

## Ops / observability — see [observability.md](observability.md)

These endpoints are **unauthenticated** — they carry no secrets
and are safe for sidecar supervisors and monitoring probes to
poll without a key. Full response schemas and semantics are in
[observability.md](observability.md); brief summary:

- `GET /v1/proxy-info` — build info, active modes, account
  snapshot. **Since** `cursor3.11/v0.2.1`.
- `GET /v1/capabilities` — compile-time feature matrix
  (streaming / caching / tools / thinking / http_version options).
  **Since** `cursor3.11/v0.2.6`.
- `GET /v1/introspect/recent-tools?since=<duration>` —
  ring-buffered snapshot of tools recently declared by clients.
  **Since** `cursor3.11/v0.2.6`.
- `GET /v1/introspect/recent-mcp-servers?since=<duration>` —
  same ring, projected onto MCP servers. **Since** `cursor3.11/v0.2.6`.

## Agent mode — see [agents.md](agents.md)

`/v1/agents/*` routes, backed by `@cursor/sdk`. Enables codebase
indexing, MCP server management, skills, hooks, cloud runtimes.
**Since** `cursor3.11/v0.3.0`.

| Route | Purpose |
|---|---|
| `POST /v1/agents` | Create an agent (local or cloud runtime) |
| `GET /v1/agents` | List agents |
| `GET /v1/agents/{id}` | Describe a single agent |
| `POST /v1/agents/{id}/runs` | Run a prompt (streaming or one-shot) |
| `DELETE /v1/agents/{id}` | Close an agent (idempotent) |

All are **auth**-gated by the same `-api-keys` middleware.

## Error envelope

Regardless of the shape the client speaks, error responses use the
same JSON envelope:

```json
{
  "error": {
    "type":    "invalid_request_error",
    "message": "…"
  }
}
```

Common `type` values:

- `authentication_error` — 401.
- `invalid_request_error` — 400 (malformed body, unsupported
  field like Anthropic server tools, unknown model,
  `Max Mode Required`).
- `forbidden_region` — 403 (upstream geo-gate; set `HTTPS_PROXY`).
- `rate_limit_error` — 429 (Cursor account rate-limited; the proxy
  does not add its own throttling).
- `upstream_error` — 502 (Cursor backend non-2xx or streaming
  trailer error mid-response).
- `service_unavailable` — 503 (wire-mode auth not loaded yet).
