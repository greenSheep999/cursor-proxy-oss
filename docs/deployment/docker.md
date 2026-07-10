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

## Verifying

```bash
curl http://localhost:8317/v1/models -H "Authorization: Bearer $SK" \
  | python3 -m json.tool
```

Response should list every model your Cursor account can use.
