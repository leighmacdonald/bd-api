package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"io"
	"log"
	"net/http"
	"time"
)

func get(ctx context.Context, url string, recv interface{}) (*http.Response, error) {
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

	if recv != nil {
		body, errRead := io.ReadAll(resp.Body)
		if errRead != nil {
			return nil, errors.Wrapf(errNewReq, "error reading stream: %v", errRead)
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				log.Panicf("Failed to close response body: %v", err)
			}
		}()
		if errUnmarshal := json.Unmarshal(body, &recv); errUnmarshal != nil {
			return resp, errors.Wrapf(errUnmarshal, "Failed to decode json: %v", errUnmarshal)
		}
	}
	return resp, nil
}

type KickEvent struct {
	SteamID    steamid.SID64 `json:"steam_id"`
	ServerName string        `json:"server_name"`
}

type UserProfile struct {
	SteamId     steamid.SID64 `json:"steam_id"`
	LogsTFCount int           `json:"logs_tf_count"`
	Seasons     []Season      `json:"seasons"`
}

func onPostKick(w http.ResponseWriter, _ *http.Request) {
	if _, errWrite := fmt.Fprintf(w, ""); errWrite != nil {
		log.Printf("failed to write response body: %v\n", errWrite)
	}
}

func sendItem(w http.ResponseWriter, req *http.Request, item any) {
	resp, jsonErr := json.Marshal(item)
	if jsonErr != nil {
		log.Printf("Failed to encode summary: %v\n", jsonErr)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	http.ServeContent(w, req, "", time.Now(), bytes.NewReader(resp))
}

func handleGetSummary() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		steamIdQuery := req.URL.Query().Get("steam_id")
		steamId, steamIdErr := steamid.SID64FromString(steamIdQuery)
		if steamIdErr != nil {
			http.Error(w, "Invalid steam id", http.StatusBadRequest)
			return
		}
		item := cache.summary.Get(steamId)
		sendItem(w, req, item.Value())
	}
}

func handleGetCompetitive() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		steamIdQuery := req.URL.Query().Get("steam_id")
		steamId, steamIdErr := steamid.SID64FromString(steamIdQuery)
		if steamIdErr != nil {
			http.Error(w, "Invalid steam id", http.StatusBadRequest)
			return
		}
		item := cache.seasons.Get(steamId)
		sendItem(w, req, item.Value())
	}
}

func handleGetBans() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		steamIdQuery := req.URL.Query().Get("steam_id")
		steamId, steamIdErr := steamid.SID64FromString(steamIdQuery)
		if steamIdErr != nil {
			http.Error(w, "Invalid steam id", http.StatusBadRequest)
			return
		}
		item := cache.bans.Get(steamId)
		sendItem(w, req, item.Value())
	}
}

func getHandler(wrappedFn func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		if req.Method != "GET" {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		wrappedFn(w, req)
	}
}
