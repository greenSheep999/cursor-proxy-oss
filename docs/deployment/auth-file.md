# Preparing a Cursor auth file

`cursor-proxy` reads a JSON file that contains the OAuth material for one
Cursor account. The exact shape is the "CPA auth file" schema (matches
CLIProxyAPI's `auths/` directory), which looks like:

```json
{
  "type": "cursor",
  "email": "you@example.com",
  "access_token": "eyJ...",
  "refresh_token": "eyJ...",
  "auth_kind": "Auth_0",
  "machine_id": "...64-hex...",
  "mac_machine_id": "...64-hex...",
  "issued_at": "2026-07-10T03:13:58Z",
  "last_refresh": "2026-07-10T03:13:58Z",
  "refreshable": true
}
```

You need **one** such file per account. Save it as
`./auth/current.json` (or whichever name you point
`CURSOR_PROXY_ACCOUNT_FILE` at).

## How to get one

You have a Cursor Pro subscription and an installed Cursor IDE. Any of
these works:

### A. Extract from the running IDE (macOS)

The Cursor IDE stores its access token in a local SQLite database. A
one-shot dumper reads that and writes the JSON shape above.

The dumper binary is not published in this repository (it's part of the
same build family that produces the runtime image, and shipping it
would leak more of the reverse-engineered surface than the runtime
alone). If you have access to it, run:

```bash
cursor-export -stdout > auth/current.json
chmod 600 auth/current.json
```

Otherwise, request the binary from the maintainer.

### B. Copy from CLIProxyAPI's `auths/` directory

If you already run CLIProxyAPI, its cursor-provider auth files live in
`auths/`. Copy the one you want:

```bash
cp ~/.cli-proxy-api/cursor-you@example.com.json \
   /path/to/cursor-proxy-oss/auth/current.json
```

### C. From another operator

If a colleague already has one of the above binaries, ask them for a
copy of the JSON. Treat it like an API key — it grants access to their
Cursor account and any usage counts against their Pro plan.

## What's inside

The two fields that matter for authentication:

- `access_token` — a Cursor JWT with a ~60-day expiry.
- `refresh_token` — used to renew the access token unattended.

The token pair is scoped to your Cursor account. Anyone who obtains the
file can make requests that count against your Pro allowance. Never
commit `auth/`. The default `.gitignore` in this repository excludes
it.

## Rotating

Cursor's access tokens expire; the proxy automatically refreshes them
using the refresh token when `refreshable: true`. If refresh fails
(revoked device, expired refresh token, IDE re-login flow), regenerate
the file with `cursor-export` and drop the new one in place. The proxy
picks it up on the next request without restart.

## Multiple accounts

Drop multiple files under `./auth/`. Point
`CURSOR_PROXY_ACCOUNT_FILE` at the one you want a given container to
use. For true round-robin/pooling across accounts, use CLIProxyAPI's
scheduler instead of `cursor-proxy` — this project is a single-account
bridge.
