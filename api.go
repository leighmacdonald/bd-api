package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/leighmacdonald/steamweb"
	"go.uber.org/zap"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

func onPostKick(w http.ResponseWriter, _ *http.Request) {
	if _, errWrite := fmt.Fprintf(w, ""); errWrite != nil {
		log.Printf("failed to write response body: %v\n", errWrite)
	}
}

func sendItem(w http.ResponseWriter, req *http.Request, item any) {
	resp, jsonErr := json.Marshal(item)
	if jsonErr != nil {
		logger.Error("Failed to marshal json item", zap.Error(jsonErr))
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	http.ServeContent(w, req, "", time.Now(), bytes.NewReader(resp))
}

func handleGetSummary() func(http.ResponseWriter, *http.Request) {
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

func handleGetBans() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		steamIDQuery := req.URL.Query().Get("steam_id")
		sid, steamIDErr := steamid.SID64FromString(steamIDQuery)
		if steamIDErr != nil {
			http.Error(w, "Invalid steam id", http.StatusBadRequest)
			return
		}
		item := cache.bans.Get(sid)
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

func loadProfile(steamID steamid.SID64, profile *Profile) error {
	var mu sync.RWMutex
	wg := &sync.WaitGroup{}
	wg.Add(7)
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
	go func() {
		defer wg.Done()
		item := cache.rglSeasons.Get(steamID)
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
	return nil
}

func handleGetProfile() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		steamIDQuery := req.URL.Query().Get("steam_id")
		steamID, steamIDErr := steamid.SID64FromString(steamIDQuery)
		if steamIDErr != nil {
			http.Error(w, "Invalid steam id", http.StatusBadRequest)
			return
		}
		var profile Profile
		if errProfile := loadProfile(steamID, &profile); errProfile != nil {
			logger.Error("Failed to load profile", zap.Error(errProfile))
			http.Error(w, "Failed to load profile", http.StatusInternalServerError)
			return
		}
		sendItem(w, req, &profile)
	}
}

const profilesSlugURL = "/profiles/"

func logHTTPErr(w http.ResponseWriter, message string, err error, statusCode int) {
	http.Error(w, "Failed to generate json", statusCode)
	logger.Error(message, zap.Error(err))
}

func handleGetProfiles(ctx context.Context) func(http.ResponseWriter, *http.Request) {
	style := styles.Get("monokai")
	if style == nil {
		style = styles.Fallback
	}
	formatter := html.New(html.WithClasses(true))
	lexer := lexers.Get("json")
	write := func(w http.ResponseWriter, format string, args ...any) bool {
		if _, errWrite := fmt.Fprintf(w, format, args...); errWrite != nil {
			logHTTPErr(w, "Failed to write response body", errWrite, http.StatusInternalServerError)
			return false
		}
		return true
	}
	return func(w http.ResponseWriter, req *http.Request) {
		var slug string
		if strings.HasPrefix(req.URL.Path, profilesSlugURL) {
			slug = req.URL.Path[len(profilesSlugURL):]
		}
		if slug == "" {
			http.Error(w, "Invalid SID", http.StatusNotFound)
			return
		}
		lCtx, cancel := context.WithTimeout(ctx, time.Second*10)
		defer cancel()
		sid64, errSid := steamid.ResolveSID64(lCtx, slug)
		if errSid != nil {
			http.Error(w, "Invalid SID", http.StatusNotFound)
		}
		w.Header().Set("Content-Type", "text/html")
		var profile Profile
		if errProfile := loadProfile(sid64, &profile); errProfile != nil {
			logHTTPErr(w, "Failed to load profile", errProfile, http.StatusInternalServerError)
			return
		}
		jsonBody, errJSON := json.MarshalIndent(profile, "", "    ")
		if errJSON != nil {
			logHTTPErr(w, "Failed to generate json", errJSON, http.StatusInternalServerError)
			return
		}
		iterator, errTokenize := lexer.Tokenise(nil, string(jsonBody))
		if errTokenize != nil {
			logHTTPErr(w, "Failed to tokenise json", errJSON, http.StatusInternalServerError)
			return
		}
		if !write(w, `<!DOCTYPE html><html><head><title>%s</title></head><body><style> body {background-color: #272822;}`, profile.Summary.PersonaName) {
			return
		}
		if errWrite := formatter.WriteCSS(w, style); errWrite != nil {
			logHTTPErr(w, "Failed to generate HTML", errWrite, http.StatusInternalServerError)
		}
		if !write(w, `</style>`) {
			return
		}
		if errFormat := formatter.Format(w, style, iterator); errFormat != nil {
			logHTTPErr(w, "Failed to format json", errFormat, http.StatusInternalServerError)
			return
		}
		write(w, `</body></html>`)
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
