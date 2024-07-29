package main

import (
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
