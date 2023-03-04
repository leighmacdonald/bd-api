package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/leighmacdonald/steamweb"
	"github.com/pkg/errors"
	"io"
	"log"
	"net/http"
	"sort"
	"sync"
	"time"
)

func get(ctx context.Context, url string, receiver interface{}) (*http.Response, error) {
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

	if receiver != nil {
		body, errRead := io.ReadAll(resp.Body)
		if errRead != nil {
			return nil, errors.Wrapf(errNewReq, "error reading stream: %v", errRead)
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				log.Panicf("Failed to close response body: %v", err)
			}
		}()
		if errUnmarshal := json.Unmarshal(body, &receiver); errUnmarshal != nil {
			return resp, errors.Wrapf(errUnmarshal, "Failed to decode json: %v", errUnmarshal)
		}
	}
	return resp, nil
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

func handleGetSummary(cache caches) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		steamIDQuery := req.URL.Query().Get("steam_id")
		steamID, steamIDErr := steamid.SID64FromString(steamIDQuery)
		if steamIDErr != nil || !steamID.Valid() {
			http.Error(w, "Invalid steam id", http.StatusBadRequest)
			return
		}
		item := cache.summary.Get(steamID)
		sendItem(w, req, item.Value())
	}
}

func handleGetBans(cache caches) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		steamIDQuery := req.URL.Query().Get("steam_id")
		steamID, steamIDErr := steamid.SID64FromString(steamIDQuery)
		if steamIDErr != nil {
			http.Error(w, "Invalid steam id", http.StatusBadRequest)
			return
		}
		item := cache.bans.Get(steamID)
		sendItem(w, req, item.Value())
	}
}

// Profile is a high level meta profile of several services
type Profile struct {
	Summary   steamweb.PlayerSummary  `json:"summary"`
	BanState  steamweb.PlayerBanState `json:"ban_state"`
	Seasons   []Season                `json:"seasons"`
	Friends   []steamweb.Friend       `json:"friends"`
	LogsCount int64                   `json:"logs_count"`
}

func handleGetProfile(cache caches) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		steamIDQuery := req.URL.Query().Get("steam_id")
		steamID, steamIDErr := steamid.SID64FromString(steamIDQuery)
		if steamIDErr != nil {
			http.Error(w, "Invalid steam id", http.StatusBadRequest)
			return
		}
		profile := Profile{}
		var mu sync.RWMutex
		wg := &sync.WaitGroup{}
		wg.Add(6)
		go func() {
			defer wg.Done()
			item := cache.bans.Get(steamID)
			if item != nil {
				profile.BanState = item.Value()
			}
		}()
		go func() {
			defer wg.Done()
			item := cache.summary.Get(steamID)
			if item != nil {
				profile.Summary = item.Value()
			}
		}()
		go func() {
			defer wg.Done()
			item := cache.logsTF.Get(steamID)
			if item != nil {
				profile.LogsCount = item.Value()
			}
		}()
		go func() {
			defer wg.Done()
			item := cache.friends.Get(steamID)
			if item != nil {
				profile.Friends = item.Value()
			}
		}()
		go func() {
			defer wg.Done()
			item := cache.etf2lSeasons.Get(steamID)
			if item != nil {
				mu.Lock()
				profile.Seasons = append(profile.Seasons, item.Value()...)
				mu.Unlock()
			}

		}()
		go func() {
			defer wg.Done()
			item := cache.ugcSeasons.Get(steamID)
			if item != nil {
				mu.Lock()
				profile.Seasons = append(profile.Seasons, item.Value()...)
				mu.Unlock()
			}
		}()
		wg.Wait()
		sort.Slice(profile.Seasons, func(i, j int) bool {
			return profile.Seasons[i].DivisionInt < profile.Seasons[j].DivisionInt
		})
		sendItem(w, req, &profile)
	}
}

func getHandler(wrappedFn func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		wrappedFn(w, req)
	}
}
