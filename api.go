package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"github.com/pkg/errors"
)

const (
	maxResults = 100

	apiTimeout = time.Second * 10
)

var (
	indexTMPL *template.Template
	encoder   *styleEncoder
)

// Profile is a high level meta profile of several services.
type Profile struct {
	Summary    steamweb.PlayerSummary  `json:"summary"`
	BanState   steamweb.PlayerBanState `json:"ban_state"`
	Seasons    []Season                `json:"seasons"`
	Friends    []steamweb.Friend       `json:"friends"`
	SourceBans []SbBanRecord           `json:"source_bans"`
	LogsCount  int64                   `json:"logs_count"`
}

func loadProfiles(ctx context.Context, database *pgStore, cache cache, steamIDs steamid.Collection) ([]Profile, error) {
	var ( //nolint:prealloc
		waitGroup  = &sync.WaitGroup{}
		summaries  []steamweb.PlayerSummary
		bans       []steamweb.PlayerBanState
		profiles   []Profile
		friends    friendMap
		sourceBans BanRecordMap
	)

	if len(steamIDs) > maxResults {
		return nil, errTooMany
	}

	localCtx, cancel := context.WithTimeout(ctx, apiTimeout)
	defer cancel()

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		sbRecords, errSB := database.sbGetBansBySID(localCtx, steamIDs)
		if errSB != nil {
			slog.Error("Failed to load sourcebans records", ErrAttr(errSB))
		}

		sourceBans = sbRecords
	}()

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		sum, errSum := getSteamSummaries(localCtx, cache, steamIDs)
		if errSum != nil || len(sum) == 0 {
			slog.Error("Failed to load player summaries", ErrAttr(errSum))
		}

		summaries = sum
	}()

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		banState, errBanState := getSteamBans(localCtx, cache, steamIDs)
		if errBanState != nil || len(banState) == 0 {
			slog.Error("Failed to load player ban states", ErrAttr(errBanState))
		}

		bans = banState
	}()

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		friends = getSteamFriends(localCtx, cache, steamIDs)
	}()

	waitGroup.Wait()

	if len(steamIDs) == 0 || len(summaries) == 0 {
		return nil, errors.New("No results fetched")
	}

	for _, sid := range steamIDs {
		var profile Profile

		for _, summary := range summaries {
			if summary.SteamID == sid {
				profile.Summary = summary

				break
			}
		}

		for _, ban := range bans {
			if ban.SteamID == sid {
				profile.BanState = ban

				break
			}
		}

		if records, ok := sourceBans[sid.String()]; ok {
			profile.SourceBans = records
		} else {
			// Dont return null json values
			profile.SourceBans = []SbBanRecord{}
		}

		if friendsList, ok := friends[sid.String()]; ok {
			profile.Friends = friendsList
		} else {
			profile.Friends = []steamweb.Friend{}
		}

		profile.Seasons = []Season{}
		sort.Slice(profile.Seasons, func(i, j int) bool {
			return profile.Seasons[i].DivisionInt < profile.Seasons[j].DivisionInt
		})
		profiles = append(profiles, profile)
	}

	return profiles, nil
}

func responseErr(w http.ResponseWriter, r *http.Request, status int, err error, userMsg string) {
	msg := err.Error()
	if userMsg != "" {
		msg = userMsg
	}
	renderResponse(w, r, status, map[string]string{
		"error": msg,
	}, &baseTmplArgs{ //nolint:exhaustruct
		Title: "Error",
	})
	slog.Error("error executing request", ErrAttr(err))
}

func responseOk(w http.ResponseWriter, r *http.Request, status int, data any, args syntaxTemplate) {
	renderResponse(w, r, status, data, args)
}

func renderResponse(w http.ResponseWriter, r *http.Request, status int, data any, args syntaxTemplate) {
	if data == nil {
		data = []string{}
	}

	jsonBuff := bytes.NewBuffer(nil)

	if err := json.NewEncoder(jsonBuff).Encode(data); err != nil {
		slog.Error("Failed to encode response", ErrAttr(err))
		responseErr(w, r, http.StatusInternalServerError, err, "encoder failed")
		return
	}

	if strings.Contains(strings.ToLower(r.Header.Get("Accept")), "text/html") {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(status)

		css, body, errEnc := encoder.Encode(jsonBuff.String())
		if errEnc != nil {
			responseErr(w, r, http.StatusInternalServerError, errEnc, "encoder failed")

			return
		}

		args.setCSS(css)
		args.setBody(body)

		if errExec := indexTMPL.Execute(w, args); errExec != nil {
			slog.Error("failed to execute template", ErrAttr(errExec))
		}

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, errWrite := fmt.Fprint(w, jsonBuff.String()); errWrite != nil {
		slog.Error("failed to write out json response", ErrAttr(errWrite))
	}
}

func steamIDFromSlug(w http.ResponseWriter, r *http.Request) (steamid.SteamID, bool) {
	sid64 := steamid.New(r.PathValue("steam_id"))
	if !sid64.Valid() {
		responseErr(w, r, http.StatusNotFound, errInvalidSteamID, "")

		return steamid.SteamID{}, false
	}

	return sid64, true
}

//func renderResponse(ctx *gin.Context, encoder *styleEncoder, value any, args syntaxTemplate) {
//	if !strings.Contains(strings.ToLower(ctx.GetHeader("Accept")), "text/html") {
//		ctx.JSON(http.StatusOK, value)
//
//		return
//	}
//
//	css, body, errEncode := encoder.Encode(value)
//	if errEncode != nil {
//		ctx.AbortWithStatusJSON(http.StatusInternalServerError, "Failed to load profile")
//
//		return
//	}
//
//	args.setCSS(css)
//	args.setBody(body)
//	ctx.HTML(http.StatusOK, "", args)
//}

func initTemplate() error {
	tmplProfiles, errTmpl := template.New("").Parse(`<!DOCTYPE html>
<html>
<head>
	<title>{{ .Title }}</title>
	<style> body {background-color: #272822;} {{ .CSS }} </style>
</head>
<body>{{ .Body }}</body>
</html>`)

	if errTmpl != nil {
		return errors.Wrap(errTmpl, "Failed to parse html template")
	}

	indexTMPL = tmplProfiles

	return nil
}

func createRouter(database *pgStore, cacheHandler cache) (*http.ServeMux, error) {
	encoder = newStyleEncoder()
	if errTmpl := initTemplate(); errTmpl != nil {
		return nil, errTmpl
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /bans", handleGetBans())
	mux.HandleFunc("GET /summary", handleGetSummary(cacheHandler))
	mux.HandleFunc("GET /profile", handleGetProfile(database, cacheHandler))
	mux.HandleFunc("GET /comp", handleGetComp(cacheHandler))
	mux.HandleFunc("GET /friends", handleGetFriendList(cacheHandler))
	mux.HandleFunc("GET /sourcebans", handleGetSourceBansMany(database))
	mux.HandleFunc("GET /sourcebans/:steam_id", handleGetSourceBans(database))
	mux.HandleFunc("GET /bd", handleGetBotDetector(database))

	return mux, nil
}

const (
	apiHandlerTimeout = 10 * time.Second

	shutdownTimeout = 10 * time.Second
)

func newHTTPServer(ctx context.Context, router *http.ServeMux, addr string) *http.Server {
	httpServer := &http.Server{ //nolint:exhaustruct
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  apiHandlerTimeout,
		WriteTimeout: apiHandlerTimeout,
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
	}

	return httpServer
}

func runHTTP(ctx context.Context, router *http.ServeMux, listenAddr string) int {
	httpServer := newHTTPServer(ctx, router, listenAddr)

	go func() {
		if errServe := httpServer.ListenAndServe(); errServe != nil && !errors.Is(errServe, http.ErrServerClosed) {
			slog.Error("error trying to shutdown http service", ErrAttr(errServe))
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if errShutdown := httpServer.Shutdown(shutdownCtx); errShutdown != nil { //nolint:contextcheck
		slog.Error("Error shutting down http service", ErrAttr(errShutdown))

		return 1
	}

	return 0
}
