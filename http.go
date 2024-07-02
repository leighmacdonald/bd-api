package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

var (
	errRequestCreate    = errors.New("failed to create request")
	errRequestPerform   = errors.New("failed to perform request")
	errResponseRead     = errors.New("failed to read response body")
	errResponseDecode   = errors.New("failed to decode response")
	errResponseJSON     = errors.New("failed to generate json response")
	errResponseTokenize = errors.New("failed to tokenize json body")
	errResponseCSS      = errors.New("failed to write css body")
	errResponseFormat   = errors.New("failed to format body")
)

// NewHTTPClient allocates a preconfigured *http.Client.

func NewHTTPClient() *http.Client {
	c := &http.Client{ //nolint:exhaustruct
		Timeout: time.Second * 10,
	}

	return c
}

func get(ctx context.Context, url string, receiver interface{}) (*http.Response, error) {
	req, errNewReq := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if errNewReq != nil {
		return nil, errors.Join(errNewReq, errRequestCreate)
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
		return nil, errors.Join(errResp, errRequestPerform)
	}

	if receiver != nil {
		defer logCloser(resp.Body)

		if errUnmarshal := json.NewDecoder(resp.Body).Decode(&receiver); errUnmarshal != nil {
			return resp, errors.Join(errUnmarshal, errResponseDecode)
		}
	}

	return resp, nil
}
