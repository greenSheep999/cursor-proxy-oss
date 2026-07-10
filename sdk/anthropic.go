package sdk

import (
	"context"
	"net/http"
)

// AnthropicMessages calls POST /v1/messages and returns the full
// Anthropic-shape response.
//
// Note on auth: Anthropic's SDK sends the key on `x-api-key`. If your
// downstream tooling expects that shape, construct the client with
// Config.AuthChannel = AuthAPIKey.
func (c *Client) AnthropicMessages(ctx context.Context, req AnthropicRequest) (*AnthropicResponse, error) {
	req.Stream = false
	httpReq, err := c.newRequest(ctx, http.MethodPost, "/v1/messages", req)
	if err != nil {
		return nil, err
	}
	// Anthropic requires this header on real calls; the proxy is
	// lenient but we set it for wire-parity with the real service.
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	out := &AnthropicResponse{}
	if err := c.do(httpReq, out); err != nil {
		return nil, err
	}
	return out, nil
}

// CountTokens calls POST /v1/messages/count_tokens and returns the
// estimated input-token count.
//
// The count is a heuristic (approximately runes/3.5) — cursor-proxy
// does not have access to Anthropic's real tokenizer. The returned
// value is safe to display as an "approximate" prompt size.
func (c *Client) CountTokens(ctx context.Context, req CountTokensRequest) (*CountTokensResponse, error) {
	httpReq, err := c.newRequest(ctx, http.MethodPost, "/v1/messages/count_tokens", req)
	if err != nil {
		return nil, err
	}
	out := &CountTokensResponse{}
	if err := c.do(httpReq, out); err != nil {
		return nil, err
	}
	return out, nil
}
