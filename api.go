package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/leighmacdonald/steamweb"
	"github.com/pkg/errors"
	"io"
	"log"
	"net/http"
	"sync"
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

func handleGetSummary(timeout time.Duration) func(http.ResponseWriter, *http.Request) {
	type cachedItem struct {
		created time.Time
		summary steamweb.PlayerSummary
	}
	cache := map[steamid.SID64]cachedItem{}
	mu := &sync.RWMutex{}
	return func(w http.ResponseWriter, req *http.Request) {
		if req.Method != "GET" {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		steamIdQuery := req.URL.Query().Get("steam_id")
		steamId, steamIdErr := steamid.SID64FromString(steamIdQuery)
		if steamIdErr != nil {
			http.Error(w, "Invalid steam id", http.StatusBadRequest)
			return
		}
		mu.RLock()
		item, found := cache[steamId]
		mu.RUnlock()
		if !found || time.Since(item.created) > timeout {
			ids := steamid.Collection{steamId}
			summaries, errSum := steamweb.PlayerSummaries(ids)
			if errSum != nil || len(ids) != len(summaries) {
				log.Printf("Failed to fetch summary: %v\n", errSum)
				http.Error(w, "Internal error", http.StatusInternalServerError)
				return
			}
			item = cachedItem{summary: summaries[0], created: time.Now()}
			mu.Lock()
			cache[steamId] = item
			mu.Unlock()
		}
		resp, jsonErr := json.Marshal(item.summary)
		if jsonErr != nil {
			log.Printf("Failed to encode summary: %v\n", jsonErr)
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		if _, errFmt := fmt.Fprint(w, resp); errFmt != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}
	}
}

func handleGetCompetitive(timeout time.Duration) func(http.ResponseWriter, *http.Request) {
	type cachedItem struct {
		created time.Time
		seasons []Season
	}
	cache := map[steamid.SID64]cachedItem{}
	mu := &sync.RWMutex{}
	return func(w http.ResponseWriter, req *http.Request) {
		if req.Method != "GET" {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		steamIdQuery := req.URL.Query().Get("steam_id")
		steamId, steamIdErr := steamid.SID64FromString(steamIdQuery)
		if steamIdErr != nil {
			http.Error(w, "Invalid steam id", http.StatusBadRequest)
			return
		}
		mu.RLock()
		item, found := cache[steamId]
		mu.RUnlock()
		if !found || time.Since(item.created) > timeout {
			seasons, errSum := fetchSeasons(steamId)
			if errSum != nil {
				log.Printf("Failed to fetch summary: %v\n", errSum)
				http.Error(w, "Internal error", http.StatusInternalServerError)
				return
			}
			item = cachedItem{seasons: seasons, created: time.Now()}
			mu.Lock()
			cache[steamId] = item
			mu.Unlock()
		}
		resp, jsonErr := json.Marshal(item.seasons)
		if jsonErr != nil {
			log.Printf("Failed to encode summary: %v\n", jsonErr)
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		if _, errFmt := fmt.Fprint(w, resp); errFmt != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}
	}
}
func handleGetBans(timeout time.Duration) func(http.ResponseWriter, *http.Request) {
	type cachedItem struct {
		created  time.Time
		banState steamweb.PlayerBanState
	}
	cache := map[steamid.SID64]cachedItem{}
	mu := &sync.RWMutex{}
	return func(w http.ResponseWriter, req *http.Request) {
		if req.Method != "GET" {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		steamIdQuery := req.URL.Query().Get("steam_id")
		steamId, steamIdErr := steamid.SID64FromString(steamIdQuery)
		if steamIdErr != nil {
			http.Error(w, "Invalid steam id", http.StatusBadRequest)
			return
		}
		mu.RLock()
		item, found := cache[steamId]
		mu.RUnlock()
		if !found || time.Since(item.created) > timeout {
			ids := steamid.Collection{steamId}
			bans, errSum := steamweb.GetPlayerBans(ids)
			if errSum != nil || len(ids) != len(bans) {
				log.Printf("Failed to fetch ban: %v\n", errSum)
				http.Error(w, "Internal error", http.StatusInternalServerError)
				return
			}
			item = cachedItem{banState: bans[0], created: time.Now()}
			mu.Lock()
			cache[steamId] = item
			mu.Unlock()
		}
		resp, jsonErr := json.Marshal(item.banState)
		if jsonErr != nil {
			log.Printf("Failed to encode ban: %v\n", jsonErr)
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		if _, errFmt := fmt.Fprint(w, resp); errFmt != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}
	}
}
