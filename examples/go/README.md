# Go examples

Small self-contained programs demonstrating the `sdk/` package.

| Directory | What it does |
|---|---|
| `chat/` | Streams a chat completion to stdout |
| `list-models/` | Prints every model the connected account can use, grouped by owner |

Run any of them with:

```bash
export CURSOR_PROXY_URL=http://localhost:8317      # optional
export CURSOR_PROXY_API_KEY=sk-cp-...              # required

go run ./examples/go/chat "hello"
go run ./examples/go/list-models
```

Each file is under 100 lines and imports only `stdlib` plus the local
`sdk` package. Copy them into your own module as a starting point.
