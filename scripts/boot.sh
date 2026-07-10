#!/usr/bin/env bash
# boot.sh — first-time helper.
#
# 1. Verifies docker is installed.
# 2. Generates a random API key and writes it to .env.
# 3. Creates the ./auth/ directory if missing.
# 4. Reminds the user to drop an auth file (see docs/deployment/auth-file.md).
# 5. Pulls the image and starts docker compose.
#
# Idempotent: safe to run multiple times. Will not overwrite an existing .env.

set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$here"

if ! command -v docker >/dev/null 2>&1; then
  echo "error: docker is not installed. See https://docs.docker.com/engine/install/" >&2
  exit 1
fi

if [[ ! -f .env ]]; then
  SK="sk-cp-$(openssl rand -hex 16)"
  cat > .env <<EOF
# Generated $(date -Iseconds)
CURSOR_PROXY_API_KEYS=$SK
EOF
  chmod 600 .env
  echo "✔ generated .env with a new API key"
  echo "  SK=$SK"
else
  SK="$(grep -E '^CURSOR_PROXY_API_KEYS=' .env | head -1 | cut -d= -f2)"
  echo "✔ using existing .env (SK=${SK:0:12}...)"
fi

mkdir -p auth
chmod 755 auth  # so a container UID != host UID can still ls it
if ! ls auth/*.json >/dev/null 2>&1; then
  cat <<'MSG'

! No auth file found in ./auth/.
  cursor-proxy needs one Cursor account JSON to run.
  See docs/deployment/auth-file.md for how to prepare one.

  Once you have it, put it at ./auth/current.json (or symlink) and run
  `docker compose up -d` yourself, or re-run this script.

MSG
  exit 0
fi

echo "✔ found auth file(s):"
# shellcheck disable=SC2012  # ls output is human-facing here, not machine-parsed
ls -la auth/*.json | sed 's/^/    /'

echo "→ pulling latest image..."
docker compose pull

echo "→ starting..."
docker compose up -d

sleep 2
echo
echo "→ verifying..."
if curl -sS -o /dev/null -w "%{http_code}\n" \
      http://localhost:8317/v1/models \
      -H "Authorization: Bearer $SK" \
      --max-time 8 | grep -q "^200$"; then
  echo "✔ cursor-proxy is up at http://localhost:8317"
  echo "  api key: $SK"
  echo
  echo "  Try:"
  echo "  curl http://localhost:8317/v1/models -H 'Authorization: Bearer $SK'"
else
  echo "✗ /v1/models did not return 200. Check logs:"
  echo "  docker compose logs cursor-proxy"
fi
