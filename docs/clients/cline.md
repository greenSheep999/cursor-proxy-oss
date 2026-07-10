# Cline (VS Code)

VS Code AI agent. Speaks OpenAI Chat Completions.

## Install

Search "Cline" in the VS Code marketplace.

## Configure

In Cline settings:

- **API Provider**: `OpenAI Compatible`
- **Base URL**: `http://localhost:8317/v1`
- **API Key**: your `$SK`
- **Model ID**: `composer-2.5` (or `claude-sonnet-5-medium`, `gpt-5.6-sol-medium`, `gemini-3.1-pro`, ...)

If you run `cursor-proxy` on a different machine, replace `localhost` with
that host's IP or tailscale name — remember to expose port 8317 first.

## Notes

- Cline sends `Authorization: Bearer $KEY` in the standard shape.
- Tool-use / diff-apply flow works — Cline receives `tool_calls` in the
  Chat Completions shape and drives edits accordingly.
- If Cline complains about unknown models, hit `GET /v1/models` first to
  see what your Cursor account can actually access:

  ```bash
  curl http://localhost:8317/v1/models -H "Authorization: Bearer $SK" \
    | python3 -m json.tool
  ```
