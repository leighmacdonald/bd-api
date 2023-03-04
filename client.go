package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/leighmacdonald/steamweb"
	"io"
	"log"
	"net/http"
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
	defer func() {
		if errClose := resp.Body.Close(); errClose != nil {
			log.Printf("Failed to close body: %v\n", errClose)
		}
	}()
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
	defer func() {
		if errClose := resp.Body.Close(); errClose != nil {
			log.Printf("Failed to close body: %v\n", errClose)
		}
	}()
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
	defer func() {
		if errClose := resp.Body.Close(); errClose != nil {
			log.Printf("Failed to close body: %v\n", errClose)
		}
	}()
	return json.Unmarshal(body, profile)
}
