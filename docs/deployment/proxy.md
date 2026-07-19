# Upstream proxy (region unlock)

Cursor's backend routes requests to Anthropic, OpenAI, and Google
directly, and it decides which providers you can reach by the **IP of
your request**. Accounts served from a CN or HK egress see this trailer
on every `claude-*`, `gpt-*`, or `gemini-*` call:

```
grpc-status: 8   (ERROR_UNSUPPORTED_REGION)
grpc-message: "This model provider is not supported in your region"
```

cursor-proxy surfaces that as `HTTP 403` with a JSON body naming the
error, so a downstream SDK does not silently see "empty content". But
the actual fix is to route the upstream call through a non-gated
egress — a US or EU HTTP/HTTPS/SOCKS5 proxy.

## What's affected

| Family | Needs upstream proxy? |
|---|---|
| Cursor-native (`composer-*`) | No — served by Cursor directly |
| xAI (`grok-*`) | No |
| Kimi, GLM | No |
| **Anthropic (`claude-*`, `claude-fable-*`, `claude-sonnet-*`, `claude-opus-*`)** | **Yes** for CN/HK egress |
| **OpenAI (`gpt-5*`, `gpt-5.5*`, `gpt-5.6-{sol,terra,luna}-*`)** | **Yes** for CN/HK egress |
| **Google (`gemini-*`)** | **Yes** for CN/HK egress |

Users in US/EU / other non-gated regions can skip this entirely.

## Configuration

Three equivalent ways, pick whichever fits the deployment:

### 1. `-upstream-proxy` flag

```bash
cursor-proxy \
  -addr=127.0.0.1:8317 \
  -api-keys=$SK \
  -upstream-proxy=http://127.0.0.1:10808
```

Accepted schemes: `http://[user:pass@]host:port`,
`https://[user:pass@]host:port`, `socks5://[user:pass@]host:port`.

### 2. `HTTPS_PROXY` / `HTTP_PROXY` environment

The Go HTTP client that talks to Cursor's backend already honours the
standard proxy env vars. The `docker-compose.yml` shipped with this
repo picks them up from your `.env`:

```
# .env
HTTPS_PROXY=http://127.0.0.1:10808
HTTP_PROXY=http://127.0.0.1:10808
NO_PROXY=localhost,127.0.0.1
```

Or set them on the container directly:

```bash
docker run \
  -e HTTPS_PROXY=socks5://user:pass@remote-host:1080 \
  -e HTTP_PROXY=socks5://user:pass@remote-host:1080 \
  -e NO_PROXY=localhost,127.0.0.1 \
  -e CURSOR_PROXY_API_KEYS=$SK \
  -v ./auth:/data/accounts:ro \
  ghcr.io/greensheep999/cursor-proxy:cursor3.11-v0.3.4
```

### 3. `CURSOR_PROXY_UPSTREAM` environment

Same effect as `HTTPS_PROXY`, kept as a dedicated name to avoid
colliding with proxies the host system already sets for other software:

```
CURSOR_PROXY_UPSTREAM=socks5://user:pass@1.2.3.4:1080
```

**Resolution order** (first non-empty wins):

`-upstream-proxy` flag → `$CURSOR_PROXY_UPSTREAM` → `$HTTPS_PROXY` →
`$https_proxy` → `$HTTP_PROXY` → `$http_proxy`.

## Verifying

After starting the proxy, look for this log line:

```
[proxy] upstream proxy: http://127.0.0.1:10808
```

Then call a gated model — a valid response means the proxy is
plumbed correctly:

```bash
curl http://localhost:8317/v1/chat/completions \
  -H "Authorization: Bearer $SK" \
  -H "content-type: application/json" \
  -d '{"model":"claude-sonnet-5-medium","messages":[{"role":"user","content":"reply PONG"}]}'
# → {"choices":[{"message":{"content":"PONG",...}}], ...}
```

If you still see:

```json
{"error":{"code":"upstream_region_gate","message":"cursor upstream: Model not available: This model provider is not supported in your region"}}
```

check that your proxy actually egresses from a non-gated region:

```bash
curl -x http://127.0.0.1:10808 https://ipinfo.io/
```

The `country` field must be one Cursor doesn't gate (US and most of
Europe work; CN/HK do not).

## Common setups

- **macOS with system proxy on**: `HTTPS_PROXY=$(scutil --proxy | grep -A1 HTTPSPort | tail -1 | awk '{print "http://127.0.0.1:"$3}')` — most GUI proxy apps listen on `127.0.0.1:7890` (Clash), `127.0.0.1:1087` (V2Ray), or `127.0.0.1:10808` (V2rayN family).
- **Kubernetes**: set `env` on the Deployment (see [`kubernetes.md`](kubernetes.md)); make sure the proxy service is reachable from the pod network.
- **Chained through a jump host**: point `HTTPS_PROXY` at the jump host's local port after `ssh -L`-forwarding it, or at a proper SOCKS5 running on the jump host.
