package main

import (
	"errors"

	"golang.org/x/time/rate"
)

var (
	errFetchMatch        = errors.New("failed to fetch rgl match via api")
	errFetchMatchInvalid = errors.New("fetched invalid rgl match via api")
	errFetchTeam         = errors.New("failed to fetch rgl team via api")
	errFetchSeason       = errors.New("failed to fetch rgl season via api")
	errFetchBans         = errors.New("failed to fetch rgl bans")
)

const (
	rglRefillRate = 0.5
	rglBucketSize = 5
)

func NewRGLLimiter() *LimiterCustom {
	return &LimiterCustom{Limiter: rate.NewLimiter(rglRefillRate, rglBucketSize)}
}
