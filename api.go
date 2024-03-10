package main

import (
	"context"
	"html/template"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"github.com/pkg/errors"
)

const (
	maxResults = 100

	apiTimeout = time.Second * 10
)

// Profile is a high level meta profile of several services.

type Profile struct {
	Summary steamweb.PlayerSummary `json:"summary"`

	BanState steamweb.PlayerBanState `json:"ban_state"`

	Seasons []Season `json:"seasons"`

	Friends []steamweb.Friend `json:"friends"`

	SourceBans []SbBanRecord `json:"source_bans"`

	LogsCount int64 `json:"logs_count"`
}

func loadProfiles(ctx context.Context, db *pgStore, cache cache, steamIDs steamid.Collection) ([]Profile, error) {
	var ( //nolint:prealloc

		waitGroup = &sync.WaitGroup{}

		summaries []steamweb.PlayerSummary

		bans []steamweb.PlayerBanState

		profiles []Profile

		friends map[steamid.SID64][]steamweb.Friend

		sourceBans BanRecordMap
	)

	if len(steamIDs) > maxResults {
		return nil, ErrTooMany
	}

	localCtx, cancel := context.WithTimeout(ctx, apiTimeout)

	defer cancel()

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		sbRecords, errSB := db.sbGetBansBySID(localCtx, steamIDs)

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

		if records, ok := sourceBans[sid]; ok {
			profile.SourceBans = records
		} else {
			// Dont return null json values

			profile.SourceBans = []SbBanRecord{}
		}

		if friendsList, ok := friends[sid]; ok {
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

func steamIDFromSlug(ctx *gin.Context) (steamid.SID64, bool) {
	sid64 := steamid.New(ctx.Param("steam_id"))

	if !sid64.Valid() {

		ctx.AbortWithStatusJSON(http.StatusNotFound, "not found")

		return "", false

	}

	return sid64, true
}

//nolint:unparam

func renderSyntax(ctx *gin.Context, encoder *styleEncoder, value any, tmpl string, args syntaxTemplate) {
	if !strings.Contains(strings.ToLower(ctx.GetHeader("Accept")), "text/html") {

		ctx.JSON(http.StatusOK, value)

		return

	}

	css, body, errEncode := encoder.Encode(value)

	if errEncode != nil {

		ctx.AbortWithStatusJSON(http.StatusInternalServerError, "Failed to load profile")

		return

	}

	args.setCSS(css)

	args.setBody(body)

	ctx.HTML(http.StatusOK, tmpl, args)
}

func apiErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		for _, ginErr := range c.Errors {
			slog.Error("Unhandled HTTP Error", ErrAttr(ginErr))
		}
	}
}

const defaultTemplate = "index"

func createRouter(runMode string, db *pgStore, c cache) (*gin.Engine, error) {
	tmplProfiles, errTmpl := template.New(defaultTemplate).Parse(`<!DOCTYPE html>

<html>

<head> 

	<title>{{ .Title }}</title>

	<style> body {background-color: #272822;} {{ .CSS }} </style>

</head>

<body>{{ .Body }}</body>

</html>`)

	if errTmpl != nil {
		return nil, errors.Wrap(errTmpl, "Failed to parse html template")
	}

	gin.SetMode(runMode)

	engine := gin.New()

	engine.SetHTMLTemplate(tmplProfiles)

	engine.Use(apiErrorHandler(), gin.Recovery())

	engine.GET("/bans", handleGetBans())

	engine.GET("/summary", handleGetSummary(c))

	engine.GET("/profile", handleGetProfile(db, c))

	engine.GET("/comp", handleGetComp(c))

	engine.GET("/friends", handleGetFriendList(c))

	engine.GET("/sourcebans", handleGetSourceBansMany(db))

	engine.GET("/sourcebans/:steam_id", handleGetSourceBans(db))

	return engine, nil
}

const (
	apiHandlerTimeout = 10 * time.Second

	shutdownTimeout = 10 * time.Second
)

func newHTTPServer(ctx context.Context, router *gin.Engine, addr string) *http.Server {
	httpServer := &http.Server{ //nolint:exhaustruct

		Addr: addr,

		Handler: router,

		ReadTimeout: apiHandlerTimeout,

		WriteTimeout: apiHandlerTimeout,
	}

	return httpServer
}

func runHTTP(ctx context.Context, router *gin.Engine, listenAddr string) int {
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
