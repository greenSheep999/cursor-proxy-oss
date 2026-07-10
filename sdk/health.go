package sdk

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// Ping performs a cheap authenticated GET against /v1/models and
// returns nil when the proxy answers with a 2xx.
//
// It's a convenience for boot scripts and readiness probes — the same
// endpoint is used because it is the smallest authenticated request
// the proxy answers.
func (c *Client) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	req, err := c.newRequest(ctx, http.MethodGet, "/v1/models", nil)
	if err != nil {
		return err
	}
	if err := c.do(req, nil); err != nil {
		return err
	}
	return nil
}

// PingResult is a richer probe answer suitable for CLI/human display.
type PingResult struct {
	OK       bool
	Latency  time.Duration
	Models   int
	AuthKind string
	Err      error
}

// Probe returns a PingResult with more context than Ping.
func (c *Client) Probe(ctx context.Context) PingResult {
	start := time.Now()
	list, err := c.ListModels(ctx)
	res := PingResult{
		Latency: time.Since(start),
	}
	if err != nil {
		res.Err = err
		var apiErr *APIError
		if errors.As(err, &apiErr) {
			res.AuthKind = fmt.Sprintf("HTTP %d", apiErr.StatusCode)
		}
		return res
	}
	res.OK = true
	res.Models = len(list.Data)
	return res
}
