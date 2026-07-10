// Example: minimal chat client against cursor-proxy.
//
// Run:
//
//	go run ./examples/go/chat "explain generics in Go in 3 sentences"
//
// Env vars:
//
//	CURSOR_PROXY_URL     (default http://localhost:8317)
//	CURSOR_PROXY_API_KEY (required)
//	MODEL                (default composer-2.5)
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/greenSheep999/cursor-proxy-oss/sdk"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: chat <prompt>")
		os.Exit(2)
	}
	prompt := strings.Join(os.Args[1:], " ")

	baseURL := envOr("CURSOR_PROXY_URL", "http://localhost:8317")
	apiKey := os.Getenv("CURSOR_PROXY_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "CURSOR_PROXY_API_KEY is required")
		os.Exit(2)
	}
	model := envOr("MODEL", "composer-2.5")

	client := sdk.NewClient(sdk.Config{
		BaseURL: baseURL,
		APIKey:  apiKey,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	stream, err := client.ChatCompletionStream(ctx, sdk.ChatRequest{
		Model:    model,
		Messages: []sdk.Message{{Role: "user", Content: prompt}},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "stream:", err)
		os.Exit(1)
	}
	defer func() { _ = stream.Close() }()

	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "recv:", err)
			os.Exit(1)
		}
		for _, ch := range chunk.Choices {
			if ch.Delta.Content != "" {
				fmt.Print(ch.Delta.Content)
			}
		}
	}
	fmt.Println()
}

func envOr(k, d string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return d
}
