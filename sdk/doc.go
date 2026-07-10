// Package sdk is a small Go client for cursor-proxy's HTTP surface.
//
// cursor-proxy exposes four wire shapes on one port — OpenAI Chat
// Completions, OpenAI Responses, Anthropic Messages, and Gemini
// generateContent. This SDK gives every one of them a typed Go
// entrypoint, with SSE streaming and the four supported auth channels
// (Authorization: Bearer, x-api-key, x-goog-api-key, ?key=).
//
// The SDK does not know anything about how cursor-proxy talks to
// Cursor's backend; it treats the proxy as an ordinary HTTP service.
// You can point it at localhost, a LAN address, a k8s Service, or a
// remote host behind TLS — it does not care.
//
// Basic usage:
//
//	client := sdk.NewClient(sdk.Config{
//	    BaseURL: "http://localhost:8317",
//	    APIKey:  os.Getenv("CURSOR_PROXY_API_KEY"),
//	})
//
//	// Non-streaming
//	resp, err := client.ChatCompletion(ctx, sdk.ChatRequest{
//	    Model: "composer-2.5",
//	    Messages: []sdk.Message{
//	        {Role: "user", Content: "explain generics in Go"},
//	    },
//	})
//	if err != nil { log.Fatal(err) }
//	fmt.Println(resp.Choices[0].Message.Content)
//
//	// Streaming
//	stream, err := client.ChatCompletionStream(ctx, sdk.ChatRequest{...})
//	defer stream.Close()
//	for {
//	    chunk, err := stream.Recv()
//	    if err == io.EOF { break }
//	    fmt.Print(chunk.Choices[0].Delta.Content)
//	}
//
// The SDK targets the public HTTP contract described in
// docs/endpoints.md; if the proxy's contract evolves, the SDK evolves
// alongside it. See the examples/ directory for end-to-end programs.
package sdk
