package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/pkg/errors"
)

const defaultHTTPTimeoutMs = 15 * 1000

// NewHTTPClient allocates a preconfigured *http.Client.
func NewHTTPClient() *http.Client {
	c := &http.Client{ //nolint:exhaustruct
		Timeout: defaultHTTPTimeoutMs,
	}

	return c
}

func get(ctx context.Context, url string, receiver interface{}) (*http.Response, error) {
	req, errNewReq := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if errNewReq != nil {
		return nil, errors.Wrapf(errNewReq, "Failed to create request: %v", errNewReq)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{ //nolint:exhaustruct
		// Don't follow redirects
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, errResp := client.Do(req)
	if errResp != nil {
		return nil, errors.Wrapf(errResp, "error during get: %v", errResp)
	}

	if receiver != nil {
		body, errRead := io.ReadAll(resp.Body)
		if errRead != nil {
			return nil, errors.Wrapf(errNewReq, "error reading stream: %v", errRead)
		}

		if errUnmarshal := json.Unmarshal(body, &receiver); errUnmarshal != nil {
			return resp, errors.Wrapf(errUnmarshal, "Failed to decode json: %v", errUnmarshal)
		}
	}

	return resp, nil
}
