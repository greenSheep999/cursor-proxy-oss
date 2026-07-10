// cpctl is a small operator CLI for cursor-proxy.
//
// It's a thin wrapper around the sdk/ package: reads config from env
// or flags, calls one endpoint, prints a human-friendly result.
//
// Verbs:
//
//	cpctl health                       # probe the proxy and print latency + model count
//	cpctl models [-o json]             # list models
//	cpctl chat "message"               # one-shot chat (composer-2.5 by default)
//	cpctl chat -m gpt-5 -s "message"   # pick a model, stream to stdout
//	cpctl count "text"                 # heuristic token count
//	cpctl keygen                       # print a fresh API key (sk-cp-<hex>)
//
// Common flags (any subcommand):
//
//	-u, --url <base>       cursor-proxy base URL   [env: CURSOR_PROXY_URL, default http://localhost:8317]
//	-k, --api-key <key>    API key                 [env: CURSOR_PROXY_API_KEY]
//	-a, --auth <channel>   bearer|x-api-key|x-goog|query (default: bearer)
//	-v, --verbose          debug logging
//
// The binary has zero third-party deps beyond the local sdk package.
package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/greenSheep999/cursor-proxy-oss/sdk"
)

func main() {
	if len(os.Args) < 2 {
		usage(os.Stderr)
		os.Exit(2)
	}
	verb, rest := os.Args[1], os.Args[2:]

	switch verb {
	case "-h", "--help", "help":
		usage(os.Stdout)
	case "health":
		mustRun(cmdHealth(rest))
	case "models":
		mustRun(cmdModels(rest))
	case "chat":
		mustRun(cmdChat(rest))
	case "count":
		mustRun(cmdCount(rest))
	case "keygen":
		mustRun(cmdKeygen(rest))
	case "version":
		fmt.Println("cpctl v0.1")
	default:
		fmt.Fprintf(os.Stderr, "unknown verb %q\n\n", verb)
		usage(os.Stderr)
		os.Exit(2)
	}
}

func usage(w *os.File) {
	fmt.Fprint(w, `cpctl — cursor-proxy operator CLI

Usage:
  cpctl <verb> [flags] [args]

Verbs:
  health          probe the proxy (auth + latency + model count)
  models          list models (add -o json for JSON)
  chat <text>     one-shot chat completion; add -s to stream
  count <text>    heuristic token count (Anthropic-style)
  keygen          print a fresh sk-cp-* API key
  version         print cpctl version
  help            show this message

Global flags (accepted by every verb):
  -u, --url <base>       cursor-proxy base URL       [env: CURSOR_PROXY_URL]
  -k, --api-key <key>    API key                     [env: CURSOR_PROXY_API_KEY]
  -a, --auth <channel>   bearer | x-api-key | x-goog | query
  -v, --verbose          extra debug output

Examples:
  cpctl health
  cpctl models -o json | jq '.data[] | .id'
  cpctl chat -m composer-2.5 "explain generics in Go" -s
`)
}

// ---------- keygen ----------

func cmdKeygen(_ []string) error {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return err
	}
	fmt.Printf("sk-cp-%s\n", hex.EncodeToString(b))
	return nil
}

// ---------- helpers ----------

// commonFlags is the small subset of flag parsing every verb needs.
// We hand-roll rather than pull cobra/urfave/cli in because the CLI's
// job is small and the repo prefers zero third-party runtime deps.
type commonFlags struct {
	baseURL     string
	apiKey      string
	authChannel sdk.AuthChannel
	verbose     bool
	positional  []string
	extra       map[string]string
}

func parseFlags(args []string, keys map[string]bool) (*commonFlags, error) {
	f := &commonFlags{
		baseURL:     envDefault("CURSOR_PROXY_URL", "http://localhost:8317"),
		apiKey:      os.Getenv("CURSOR_PROXY_API_KEY"),
		authChannel: sdk.AuthBearer,
		extra:       map[string]string{},
	}
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "-u", "--url":
			i++
			if i >= len(args) {
				return nil, fmt.Errorf("%s: missing value", a)
			}
			f.baseURL = args[i]
		case "-k", "--api-key":
			i++
			if i >= len(args) {
				return nil, fmt.Errorf("%s: missing value", a)
			}
			f.apiKey = args[i]
		case "-a", "--auth":
			i++
			if i >= len(args) {
				return nil, fmt.Errorf("%s: missing value", a)
			}
			ch, err := parseAuthChannel(args[i])
			if err != nil {
				return nil, err
			}
			f.authChannel = ch
		case "-v", "--verbose":
			f.verbose = true
		case "-h", "--help":
			return nil, errHelp
		default:
			if strings.HasPrefix(a, "-") {
				key := strings.TrimLeft(a, "-")
				if !keys[key] {
					return nil, fmt.Errorf("unknown flag: %s", a)
				}
				i++
				if i >= len(args) {
					return nil, fmt.Errorf("%s: missing value", a)
				}
				f.extra[key] = args[i]
				continue
			}
			f.positional = append(f.positional, a)
		}
	}
	return f, nil
}

func parseAuthChannel(s string) (sdk.AuthChannel, error) {
	switch strings.ToLower(s) {
	case "bearer", "":
		return sdk.AuthBearer, nil
	case "x-api-key", "apikey", "anthropic":
		return sdk.AuthAPIKey, nil
	case "x-goog", "x-goog-api-key", "gemini":
		return sdk.AuthGoogAPIKey, nil
	case "query", "?key":
		return sdk.AuthQueryKey, nil
	}
	return 0, fmt.Errorf("unknown auth channel: %s", s)
}

var errHelp = fmt.Errorf("help requested")

func newClient(f *commonFlags) *sdk.Client {
	return sdk.NewClient(sdk.Config{
		BaseURL:     f.baseURL,
		APIKey:      f.apiKey,
		AuthChannel: f.authChannel,
	})
}

func envDefault(k, d string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return d
}

func mustRun(err error) {
	if err == nil {
		return
	}
	if err == errHelp {
		return
	}
	fmt.Fprintln(os.Stderr, "cpctl:", err)
	os.Exit(1)
}

// prettyPrint dumps v as indented JSON to stdout.
func prettyPrint(v any) {
	buf, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "cpctl: marshal:", err)
		return
	}
	fmt.Println(string(buf))
}
