package sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// AuthChannel selects which HTTP header (or URL parameter) the client
// uses to present the API key. cursor-proxy accepts all four in a
// fixed priority order, but different upstream SDKs prefer different
// channels — this enum lets the caller mimic whichever one their
// deployment expects.
type AuthChannel int

const (
	// AuthBearer sends "Authorization: Bearer <key>" (OpenAI SDK default).
	AuthBearer AuthChannel = iota
	// AuthAPIKey sends "x-api-key: <key>" (Anthropic SDK default).
	AuthAPIKey
	// AuthGoogAPIKey sends "x-goog-api-key: <key>" (Gemini SDK default).
	AuthGoogAPIKey
	// AuthQueryKey appends "?key=<key>" to the URL (Gemini SDK fallback).
	AuthQueryKey
)

// Config configures a Client. Only BaseURL and APIKey are required.
type Config struct {
	// BaseURL is the address cursor-proxy is listening on, without the
	// trailing "/v1" — e.g. "http://localhost:8317".
	BaseURL string

	// APIKey is one of the keys registered in CURSOR_PROXY_API_KEYS.
	APIKey string

	// AuthChannel picks the header/param used to carry the key.
	// Defaults to AuthBearer if left zero.
	AuthChannel AuthChannel

	// HTTPClient overrides the underlying transport. Leave nil to
	// use a sensible default (no request timeout so streaming works).
	HTTPClient *http.Client

	// UserAgent overrides the User-Agent header on outbound calls.
	UserAgent string
}

// Client is a lightweight HTTP client for cursor-proxy.
type Client struct {
	cfg  Config
	http *http.Client
}

// NewClient constructs a Client. It performs no network I/O.
func NewClient(cfg Config) *Client {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "http://localhost:8317"
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	if cfg.UserAgent == "" {
		cfg.UserAgent = "cursor-proxy-sdk-go/0.1"
	}
	hc := cfg.HTTPClient
	if hc == nil {
		// Deliberately no top-level timeout — the SDK is used for
		// streaming responses that can legitimately run for many
		// minutes. Callers who need a wall clock should pass a
		// context.WithTimeout to each call.
		hc = &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:        16,
				MaxIdleConnsPerHost: 8,
				IdleConnTimeout:     90 * time.Second,
			},
		}
	}
	return &Client{cfg: cfg, http: hc}
}

// BaseURL returns the configured base URL (trimmed, no trailing slash).
func (c *Client) BaseURL() string { return c.cfg.BaseURL }

// newRequest builds an authenticated HTTP request for path. If body is
// nil, no body is sent; otherwise the body is JSON-encoded.
func (c *Client) newRequest(ctx context.Context, method, path string, body any) (*http.Request, error) {
	fullURL, err := c.buildURL(path)
	if err != nil {
		return nil, err
	}
	var rdr io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("sdk: marshal body: %w", err)
		}
		rdr = bytes.NewReader(buf)
	}
	req, err := http.NewRequestWithContext(ctx, method, fullURL, rdr)
	if err != nil {
		return nil, err
	}
	if rdr != nil {
		req.Header.Set("content-type", "application/json")
	}
	req.Header.Set("user-agent", c.cfg.UserAgent)
	c.applyAuth(req)
	return req, nil
}

// buildURL joins the client's BaseURL with a path and appends the
// key query parameter when AuthChannel == AuthQueryKey.
func (c *Client) buildURL(path string) (string, error) {
	u, err := url.Parse(c.cfg.BaseURL + path)
	if err != nil {
		return "", fmt.Errorf("sdk: invalid URL: %w", err)
	}
	if c.cfg.AuthChannel == AuthQueryKey && c.cfg.APIKey != "" {
		q := u.Query()
		q.Set("key", c.cfg.APIKey)
		u.RawQuery = q.Encode()
	}
	return u.String(), nil
}

// applyAuth writes the auth header for the configured channel. It is a
// no-op when APIKey is empty (useful for hitting endpoints that do not
// require a key on setups without CURSOR_PROXY_API_KEYS).
func (c *Client) applyAuth(req *http.Request) {
	if c.cfg.APIKey == "" {
		return
	}
	switch c.cfg.AuthChannel {
	case AuthAPIKey:
		req.Header.Set("x-api-key", c.cfg.APIKey)
	case AuthGoogAPIKey:
		req.Header.Set("x-goog-api-key", c.cfg.APIKey)
	case AuthQueryKey:
		// Already appended in buildURL.
	default: // AuthBearer
		req.Header.Set("authorization", "Bearer "+c.cfg.APIKey)
	}
}

// do executes req and returns the response body, closing it on error.
// When the response status is not 2xx it decodes the body into an
// APIError and returns that instead.
func (c *Client) do(req *http.Request, out any) error {
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("sdk: http do: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode/100 != 2 {
		return decodeAPIError(resp)
	}
	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("sdk: decode response: %w", err)
	}
	return nil
}

// APIError is the shape cursor-proxy returns on non-2xx responses. It
// matches OpenAI's envelope, which is also what Anthropic /
// Gemini-shape clients tolerate when they see a 4xx.
type APIError struct {
	StatusCode int    `json:"-"`
	Code       string `json:"code,omitempty"`
	Type       string `json:"type,omitempty"`
	Message    string `json:"message,omitempty"`
	Param      any    `json:"param,omitempty"`
	// Raw carries the response body verbatim when decoding fails.
	Raw []byte `json:"-"`
}

// Error implements the error interface.
func (e *APIError) Error() string {
	switch {
	case e.Message != "" && e.Code != "":
		return fmt.Sprintf("cursor-proxy: HTTP %d %s: %s", e.StatusCode, e.Code, e.Message)
	case e.Message != "":
		return fmt.Sprintf("cursor-proxy: HTTP %d: %s", e.StatusCode, e.Message)
	default:
		return fmt.Sprintf("cursor-proxy: HTTP %d", e.StatusCode)
	}
}

// IsUnauthorized returns true when the underlying error is a 401.
func IsUnauthorized(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusUnauthorized
}

// IsNotFound returns true when the underlying error is a 404.
func IsNotFound(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound
}

func decodeAPIError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	apiErr := &APIError{StatusCode: resp.StatusCode, Raw: body}
	// The proxy nests the fields under {"error": {...}} to match the
	// OpenAI shape.
	var envelope struct {
		Error APIError `json:"error"`
	}
	if err := json.Unmarshal(body, &envelope); err == nil && envelope.Error.Message != "" {
		apiErr.Code = envelope.Error.Code
		apiErr.Type = envelope.Error.Type
		apiErr.Message = envelope.Error.Message
		apiErr.Param = envelope.Error.Param
		return apiErr
	}
	// Fall back to a plain shape (e.g. `{"error": "msg"}`).
	var plain struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &plain); err == nil && plain.Error != "" {
		apiErr.Message = plain.Error
	}
	return apiErr
}
