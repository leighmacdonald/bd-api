package main

import (
	"context"
	"encoding/json"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"io"
	"net/http"
	"time"
)

func get(ctx context.Context, url string, receiver interface{}) (*http.Response, error) {
	t0 := time.Now()
	logger.Debug("Making request", zap.String("method", http.MethodGet), zap.String("url", url))
	req, errNewReq := http.NewRequestWithContext(ctx, "GET", url, nil)
	if errNewReq != nil {
		return nil, errors.Wrapf(errNewReq, "Failed to create request: %v", errNewReq)
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{
		// Don't follow redirects
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, errResp := client.Do(req)
	if errResp != nil {
		return nil, errors.Wrapf(errResp, "error during get: %v", errResp)
	}
	logger.Debug("Request complete", zap.String("url", url), zap.Duration("time", time.Since(t0)))
	if receiver != nil {
		body, errRead := io.ReadAll(resp.Body)
		if errRead != nil {
			return nil, errors.Wrapf(errNewReq, "error reading stream: %v", errRead)
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				logger.Error("Failed to close response body", zap.Error(err))
			}
		}()
		if errUnmarshal := json.Unmarshal(body, &receiver); errUnmarshal != nil {
			return resp, errors.Wrapf(errUnmarshal, "Failed to decode json: %v", errUnmarshal)
		}
	}
	return resp, nil
}
