package main

import (
	"errors"
)

var (
	errFetchMatch        = errors.New("failed to fetch rgl match via api")
	errFetchMatchInvalid = errors.New("fetched invalid rgl match via api")
	errFetchTeam         = errors.New("failed to fetch rgl team via api")
	errFetchSeason       = errors.New("failed to fetch rgl season via api")
	errFetchBans         = errors.New("failed to fetch rgl bans")
)
