package sdk

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// ListModels calls GET /v1/models and returns the full catalogue.
func (c *Client) ListModels(ctx context.Context) (*ModelList, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/v1/models", nil)
	if err != nil {
		return nil, err
	}
	out := &ModelList{}
	if err := c.do(req, out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetModel calls GET /v1/models/{id} and returns the single-model
// detail, or an APIError with StatusCode == 404 when the ID is not in
// the catalogue.
func (c *Client) GetModel(ctx context.Context, id string) (*Model, error) {
	if id == "" {
		return nil, fmt.Errorf("sdk: GetModel: id is empty")
	}
	// URL-escape the ID so a model like "gpt-5/mini" survives round-trip.
	path := "/v1/models/" + url.PathEscape(id)
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	out := &Model{}
	if err := c.do(req, out); err != nil {
		return nil, err
	}
	return out, nil
}
