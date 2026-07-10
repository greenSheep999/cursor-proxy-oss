package sdk

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ChatCompletion calls POST /v1/chat/completions and returns the full
// response. It forces stream=false; use ChatCompletionStream for SSE.
func (c *Client) ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	req.Stream = false
	httpReq, err := c.newRequest(ctx, http.MethodPost, "/v1/chat/completions", req)
	if err != nil {
		return nil, err
	}
	out := &ChatResponse{}
	if err := c.do(httpReq, out); err != nil {
		return nil, err
	}
	return out, nil
}

// ChatCompletionStream calls POST /v1/chat/completions with stream=true
// and returns a Stream that emits ChatStreamChunk values until the
// server terminates the SSE stream. Callers MUST call Close on the
// returned stream.
func (c *Client) ChatCompletionStream(ctx context.Context, req ChatRequest) (*ChatStream, error) {
	req.Stream = true
	httpReq, err := c.newRequest(ctx, http.MethodPost, "/v1/chat/completions", req)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("accept", "text/event-stream")
	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("sdk: http do: %w", err)
	}
	if resp.StatusCode/100 != 2 {
		defer func() { _ = resp.Body.Close() }()
		return nil, decodeAPIError(resp)
	}
	return &ChatStream{body: resp.Body, reader: bufio.NewReader(resp.Body)}, nil
}

// ChatStream is a reader over server-sent Chat Completions events.
//
// The underlying protocol is OpenAI's standard: each event is a
// `data: {...}` line, terminated by `data: [DONE]`. Recv decodes one
// chunk per call; io.EOF signals end-of-stream.
type ChatStream struct {
	body   io.ReadCloser
	reader *bufio.Reader
	done   bool
}

// Recv returns the next chunk, or io.EOF when the stream ends. It is
// safe to call multiple times after io.EOF (returns io.EOF each time).
func (s *ChatStream) Recv() (*ChatStreamChunk, error) {
	if s.done {
		return nil, io.EOF
	}
	for {
		line, err := s.reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				s.done = true
				return nil, io.EOF
			}
			return nil, fmt.Errorf("sdk: read stream: %w", err)
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// SSE comments start with ":".
		if strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			// Some servers emit "event: <name>" lines before their
			// data; we ignore those and read the next line.
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "[DONE]" {
			s.done = true
			return nil, io.EOF
		}
		var chunk ChatStreamChunk
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			return nil, fmt.Errorf("sdk: decode SSE data: %w (raw=%q)", err, payload)
		}
		return &chunk, nil
	}
}

// Close closes the underlying HTTP body. Must be called by the caller
// even when Recv returns io.EOF, to release the connection.
func (s *ChatStream) Close() error {
	s.done = true
	if s.body != nil {
		return s.body.Close()
	}
	return nil
}
