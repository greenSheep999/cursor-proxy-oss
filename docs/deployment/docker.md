# Docker deployment

## Prerequisites

- Docker 20+
- A Cursor auth JSON file (see [auth-file.md](auth-file.md))

## Quick start

```bash
git clone https://github.com/greenSheep999/cursor-proxy-oss
cd cursor-proxy-oss

mkdir auth
# Drop your cursor-<email>.json here, rename or symlink to current.json
cp /path/to/cursor-me@example.com.json auth/current.json

# Generate an API key
SK=sk-cp-$(openssl rand -hex 16)
echo "CURSOR_PROXY_API_KEYS=$SK" > .env
echo "Your SK: $SK"

docker compose up -d
```

## Multi-account pool

If you have several Cursor accounts, drop **all** their JSON files
into `./auth/`. Point `CURSOR_PROXY_ACCOUNT_FILE` at the one you want
to use:

```yaml
# docker-compose.override.yml
services:
  cursor-proxy:
    environment:
      CURSOR_PROXY_ACCOUNT_FILE: /data/accounts/cursor-me@example.com.json
```

For true round-robin over a pool, use the CLIProxyAPI plugin route
instead — that project ships an account scheduler, cooling, and a web
management UI. `cursor-proxy` on its own is a single-account bridge.

## Multiple keys / multiple clients

`CURSOR_PROXY_API_KEYS` is a comma-separated allowlist. Rotate keys by
adding a new one to the list, distributing it, then removing the old:

```
CURSOR_PROXY_API_KEYS=sk-cp-old-abc,sk-cp-new-def
```

## Exposing on the LAN

By default the port is bound to `127.0.0.1`. Edit `docker-compose.yml`:

```yaml
ports:
  - "8317:8317"   # was "127.0.0.1:8317:8317"
```

And set `CURSOR_PROXY_API_KEYS` to something impossible to guess.
Consider fronting with Caddy / Traefik for TLS.

## Upgrading

```bash
docker compose pull && docker compose up -d
```

`pull_policy: always` on the compose service means restart-in-place
pulls the newest `:latest`.

## Unlocking Claude / GPT / Gemini

If you sit behind a CN or HK IP, Cursor's backend gates the Anthropic /
OpenAI / Google model families and cursor-proxy will surface HTTP 403
`Model not available in your region` for them. Cursor-native models
(`composer-*`, `grok-*`, `kimi-*`, `glm-*`) are unaffected.

To unlock the gated families, add an outbound proxy to `.env`:

```
HTTPS_PROXY=http://127.0.0.1:10808          # example: local system proxy
# or:
HTTPS_PROXY=socks5://user:pass@remote:1080  # example: remote SOCKS5
NO_PROXY=localhost,127.0.0.1
```

`docker-compose up -d` picks these up from `.env` automatically. Look
for this line in `docker compose logs cursor-proxy`:

```
[proxy] upstream proxy: http://127.0.0.1:10808
```

If your proxy runs on the host at `127.0.0.1`, containers cannot reach
it as `127.0.0.1` (that's the container's own loopback). Use one of:

- `HTTPS_PROXY=http://host.docker.internal:10808` on Docker Desktop
  (macOS/Windows).
- Run the proxy service in the same compose file with a service name.
- Bind the proxy to `0.0.0.0` on the host and use the LAN IP.

See [`proxy.md`](proxy.md) for a full reference.

## Verifying

```bash
curl http://localhost:8317/v1/models -H "Authorization: Bearer $SK" \
  | python3 -m json.tool
```

Response should list every model your Cursor account can use.

Try a Claude model to confirm the outbound proxy is wired:

```bash
curl http://localhost:8317/v1/chat/completions \
  -H "Authorization: Bearer $SK" -H "content-type: application/json" \
  -d '{"model":"claude-sonnet-5-medium","messages":[{"role":"user","content":"hi"}]}'
```
