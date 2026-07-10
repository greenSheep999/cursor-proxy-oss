# Gemini CLI

Google's `@google/gemini-cli`. Speaks the `v1beta` `generateContent`
shape.

## Install

```bash
npm install -g @google/gemini-cli
```

## Configure

```bash
export GEMINI_API_URL=http://localhost:8317
export GEMINI_API_KEY=$SK
```

Or use the query-param key that some Gemini SDKs prefer:

```bash
gemini --api-url "http://localhost:8317/v1beta/models/gemini-2.5-pro:generateContent?key=$SK"
```

`cursor-proxy` accepts the key on all of `x-goog-api-key`,
`Authorization: Bearer`, and `?key=` — pick whichever matches your SDK
version.

## Recommended models

- `gemini-3.1-pro` — best default
- `gemini-3-flash` — fastest
- `gemini-3.5-flash` — mid-tier
- `gemini-2.5-flash` — previous generation, cheapest

## Try

```bash
gemini "summarise this diff:" < <(git diff HEAD~1)
```

## Notes

- `cursor-proxy`'s `v1beta/models` list contains **all** models your
  Cursor account can use (not just Gemini ones), because client-side
  model whitelists would otherwise refuse non-Gemini names outright.
- Tool calls in Gemini format are mapped in and out (`functionCall`,
  `functionResponse` parts). If your prompt supplies function
  declarations without a name they are dropped rather than 400'ing the
  whole request.
- The stream shape is `data: {json}\n\n` with a terminal
  `finishReason:"STOP"` chunk carrying full `usageMetadata`.
- **Region gate**: from a CN/HK egress, `gemini-*` returns HTTP 403
  `Model not available in your region`. Set `-upstream-proxy` (or
  `HTTPS_PROXY`) on the cursor-proxy container to a US/EU proxy. See
  [`../deployment/proxy.md`](../deployment/proxy.md).
