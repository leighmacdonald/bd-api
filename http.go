package main

import (
	"errors"
	"net/http"
	"net/url"
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

const (
	httpTimeout = time.Second * 15
)

// NewHTTPClient allocates a preconfigured *http.Client.

func NewHTTPClient() *http.Client {
	return &http.Client{Timeout: httpTimeout}
}

func NewHTTPClientWithSwitcher() *http.Client {
	c := &http.Client{ //nolint:exhaustruct
		Timeout: httpTimeout,
		Transport: &http.Transport{
			Proxy: proxySwitcher(),
		},
	}

	return c
}

func proxySwitcher() func(r *http.Request) (*url.URL, error) {
	return func(r *http.Request) (*url.URL, error) {
		panic("fix")
		return nil, nil
	}
}
