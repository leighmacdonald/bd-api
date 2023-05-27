package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/leighmacdonald/steamweb"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"io"
	"net/http"
	"time"
)

// Client is a simple api client
type Client struct {
	*http.Client
	endPoint string
}

// NewClient returns a configured api client
func NewClient(endPoint string) *Client {
	c := Client{Client: &http.Client{}, endPoint: endPoint}
	return &c
}

// PlayerSummary fetches and returns the steam web profile summary from valves api
func (c *Client) PlayerSummary(ctx context.Context, steamID steamid.SID64, summary *steamweb.PlayerSummary) error {
	req, errReq := http.NewRequestWithContext(ctx, http.MethodGet, c.endPoint+fmt.Sprintf("/summary?steam_id=%d", steamID), nil)
	if errReq != nil {
		return errReq
	}
	resp, errResp := c.Do(req)
	if errResp != nil {
		return errResp
	}
	body, errBody := io.ReadAll(resp.Body)
	if errBody != nil {
		return errBody
	}
	defer logCloser(resp.Body)
	return json.Unmarshal(body, summary)
}

// GetPlayerBan fetches and returns the steam web ban summary from valves api
func (c *Client) GetPlayerBan(ctx context.Context, steamID steamid.SID64, banState *steamweb.PlayerBanState) error {
	req, errReq := http.NewRequestWithContext(ctx, http.MethodGet, c.endPoint+fmt.Sprintf("/bans?steam_id=%d", steamID), nil)
	if errReq != nil {
		return errReq
	}
	resp, errResp := c.Do(req)
	if errResp != nil {
		return errResp
	}
	body, errBody := io.ReadAll(resp.Body)
	if errBody != nil {
		return errBody
	}
	defer logCloser(resp.Body)
	return json.Unmarshal(body, banState)
}

// GetProfile assembles and returns a high level profile
func (c *Client) GetProfile(ctx context.Context, steamID steamid.SID64, profile *Profile) error {
	req, errReq := http.NewRequestWithContext(ctx, http.MethodGet, c.endPoint+fmt.Sprintf("/profile?steam_id=%d", steamID), nil)
	if errReq != nil {
		return errReq
	}
	resp, errResp := c.Do(req)
	if errResp != nil {
		return errResp
	}
	body, errBody := io.ReadAll(resp.Body)
	if errBody != nil {
		return errBody
	}
	defer logCloser(resp.Body)
	return json.Unmarshal(body, profile)
}

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
