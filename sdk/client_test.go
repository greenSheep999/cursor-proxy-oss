package sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestServer(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c := NewClient(Config{BaseURL: srv.URL, APIKey: "sk-test"})
	return c, srv
}

func TestApplyAuth_Bearer(t *testing.T) {
	c := NewClient(Config{BaseURL: "http://x", APIKey: "sk-abc"})
	req, err := http.NewRequest("GET", "http://x/v1/models", nil)
	if err != nil {
		t.Fatal(err)
	}
	c.applyAuth(req)
	if got := req.Header.Get("Authorization"); got != "Bearer sk-abc" {
		t.Fatalf("Authorization = %q", got)
	}
}

func TestApplyAuth_APIKey(t *testing.T) {
	c := NewClient(Config{BaseURL: "http://x", APIKey: "sk-abc", AuthChannel: AuthAPIKey})
	req, _ := http.NewRequest("GET", "http://x/v1/messages", nil)
	c.applyAuth(req)
	if got := req.Header.Get("x-api-key"); got != "sk-abc" {
		t.Fatalf("x-api-key = %q", got)
	}
	if got := req.Header.Get("Authorization"); got != "" {
		t.Fatalf("Authorization unexpectedly set: %q", got)
	}
}

func TestBuildURL_QueryKey(t *testing.T) {
	c := NewClient(Config{
		BaseURL:     "http://x",
		APIKey:      "sk-abc",
		AuthChannel: AuthQueryKey,
	})
	got, err := c.buildURL("/v1beta/models/x:generateContent")
	if err != nil {
		t.Fatal(err)
	}
	if want := "http://x/v1beta/models/x:generateContent?key=sk-abc"; got != want {
		t.Fatalf("URL = %q, want %q", got, want)
	}
}

func TestListModels(t *testing.T) {
	c, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk-test" {
			t.Fatalf("auth = %q", got)
		}
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","data":[
			{"id":"composer-2.5","object":"model","owned_by":"cursor"},
			{"id":"claude-4.5-sonnet","object":"model","owned_by":"cursor"}
		]}`))
	})
	list, err := c.ListModels(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Data) != 2 || list.Data[0].ID != "composer-2.5" {
		t.Fatalf("bad list: %+v", list)
	}
}

func TestGetModel_NotFound(t *testing.T) {
	c, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = w.Write([]byte(`{"error":{"code":"model_not_found","message":"no such model"}}`))
	})
	_, err := c.GetModel(context.Background(), "does-not-exist")
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsNotFound(err) {
		t.Fatalf("expected IsNotFound, got %v", err)
	}
}

func TestChatCompletion_HappyPath(t *testing.T) {
	c, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("wrong request: %s %s", r.Method, r.URL.Path)
		}
		var body ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Model != "composer-2.5" || len(body.Messages) != 1 {
			t.Fatalf("bad body: %+v", body)
		}
		if body.Stream {
			t.Fatalf("stream should be false for ChatCompletion")
		}
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"chatcmpl-x",
			"object":"chat.completion",
			"created":1,
			"model":"composer-2.5",
			"choices":[{"index":0,"message":{"role":"assistant","content":"HELLO"},"finish_reason":"stop"}],
			"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}
		}`))
	})
	resp, err := c.ChatCompletion(context.Background(), ChatRequest{
		Model:    "composer-2.5",
		Messages: []Message{{Role: "user", Content: "say hi"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := resp.Choices[0].Message.Content; got != "HELLO" {
		t.Fatalf("content = %v", got)
	}
	if resp.Usage.TotalTokens != 2 {
		t.Fatalf("usage = %+v", resp.Usage)
	}
}

func TestChatCompletion_Unauthorized(t *testing.T) {
	c, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		_, _ = w.Write([]byte(`{"error":{"code":"invalid_api_key","message":"nope","type":"invalid_request_error"}}`))
	})
	_, err := c.ChatCompletion(context.Background(), ChatRequest{Model: "x"})
	if err == nil {
		t.Fatal("expected 401 error")
	}
	if !IsUnauthorized(err) {
		t.Fatalf("expected IsUnauthorized true, got %v", err)
	}
}

func TestChatCompletionStream_Frames(t *testing.T) {
	c, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		frames := []string{
			`{"choices":[{"index":0,"delta":{"role":"assistant"}}],"id":"a","object":"chat.completion.chunk","model":"x","created":0}`,
			`{"choices":[{"index":0,"delta":{"content":"hi"}}],"id":"a","object":"chat.completion.chunk","model":"x","created":0}`,
			`{"choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"id":"a","object":"chat.completion.chunk","model":"x","created":0}`,
		}
		for _, f := range frames {
			_, _ = io.WriteString(w, "data: "+f+"\n\n")
			if flusher != nil {
				flusher.Flush()
			}
		}
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	})
	stream, err := c.ChatCompletionStream(context.Background(), ChatRequest{Model: "x"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = stream.Close() }()
	var content bytes.Buffer
	var finish string
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if len(chunk.Choices) > 0 {
			content.WriteString(chunk.Choices[0].Delta.Content)
			if chunk.Choices[0].FinishReason != "" {
				finish = chunk.Choices[0].FinishReason
			}
		}
	}
	if content.String() != "hi" || finish != "stop" {
		t.Fatalf("content=%q finish=%q", content.String(), finish)
	}
}

func TestChatCompletionStream_IgnoresEventLines(t *testing.T) {
	c, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "text/event-stream")
		_, _ = io.WriteString(w,
			"event: response.created\n"+
				`data: {"choices":[{"index":0,"delta":{"content":"ok"}}],"id":"a","object":"chat.completion.chunk","model":"x","created":0}`+"\n\n"+
				"data: [DONE]\n\n")
	})
	stream, err := c.ChatCompletionStream(context.Background(), ChatRequest{Model: "x"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = stream.Close() }()
	chunk, err := stream.Recv()
	if err != nil {
		t.Fatalf("first Recv: %v", err)
	}
	if chunk.Choices[0].Delta.Content != "ok" {
		t.Fatalf("expected 'ok', got %+v", chunk.Choices)
	}
	if _, err := stream.Recv(); err != io.EOF {
		t.Fatalf("expected EOF, got %v", err)
	}
}

func TestCountTokens(t *testing.T) {
	c, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages/count_tokens" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{"input_tokens": 42}`))
	})
	out, err := c.CountTokens(context.Background(), CountTokensRequest{
		Model: "claude-4.5-sonnet",
		Messages: []AnthropicMsg{
			{Role: "user", Content: "hi"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.InputTokens != 42 {
		t.Fatalf("input_tokens = %d", out.InputTokens)
	}
}

func TestProbe_Latency(t *testing.T) {
	c, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"a","object":"model"}]}`))
	})
	res := c.Probe(context.Background())
	if !res.OK {
		t.Fatalf("Probe not OK: %+v", res)
	}
	if res.Models != 1 {
		t.Fatalf("Models = %d", res.Models)
	}
	if res.Latency <= 0 {
		t.Fatalf("Latency not measured: %v", res.Latency)
	}
}

func TestProbe_Error(t *testing.T) {
	c, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		_, _ = w.Write([]byte(`{"error":{"code":"invalid_api_key","message":"nope"}}`))
	})
	res := c.Probe(context.Background())
	if res.OK {
		t.Fatal("expected not OK")
	}
	if !strings.HasPrefix(res.AuthKind, "HTTP 401") {
		t.Fatalf("AuthKind = %q", res.AuthKind)
	}
}
