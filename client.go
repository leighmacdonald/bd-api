package main

import (
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/leighmacdonald/steamweb"
	"net/http"
)

type Client struct {
	*http.Client
}

func NewClient() *Client {
	c := Client{Client: &http.Client{}}
	return &c
}

func (c *Client) PlayerSummaries(steamIDs steamid.Collection) ([]steamweb.PlayerSummary, error) {
	return nil, nil
}

func (c *Client) GetPlayerBans(steamIDs steamid.Collection) ([]steamweb.PlayerBanState, error) {
	return nil, nil
}

func (c *Client) GetFriendList(steamID steamid.SID64) ([]steamweb.Friend, error) {
	return nil, nil
}
