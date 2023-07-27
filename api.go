package main

import (
	"context"
	"html/template"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/bd-api/models"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const (
	maxResults = 100
	apiTimeout = time.Second * 10
)

// Profile is a high level meta profile of several services.
type Profile struct {
	Summary    steamweb.PlayerSummary  `json:"summary"`
	BanState   steamweb.PlayerBanState `json:"ban_state"`
	Seasons    []Season                `json:"seasons"`
	Friends    []steamweb.Friend       `json:"friends"`
	SourceBans []models.SbBanRecord    `json:"source_bans"`
	LogsCount  int64                   `json:"logs_count"`
}

func (a *App) loadProfiles(ctx context.Context, steamIDs steamid.Collection) ([]Profile, error) {
	var ( //nolint:prealloc
		waitGroup  = &sync.WaitGroup{}
		summaries  []steamweb.PlayerSummary
		bans       []steamweb.PlayerBanState
		profiles   []Profile
		friends    map[steamid.SID64][]steamweb.Friend
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

		sbRecords, errSB := a.db.sbGetBansBySID(localCtx, steamIDs)
		if errSB != nil {
			a.log.Error("Failed to load sourcebans records", zap.Error(errSB))
		}

		sourceBans = sbRecords
	}()

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		sum, errSum := a.getSteamSummaries(localCtx, steamIDs)
		if errSum != nil || len(sum) == 0 {
			a.log.Error("Failed to load player summaries", zap.Error(errSum))
		}

		summaries = sum
	}()

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		banState, errBanState := a.getSteamBans(localCtx, steamIDs)
		if errBanState != nil || len(banState) == 0 {
			a.log.Error("Failed to load player ban states", zap.Error(errBanState))
		}

		bans = banState
	}()

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		friends = a.getSteamFriends(localCtx, steamIDs)
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
			profile.SourceBans = []models.SbBanRecord{}
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

func apiErrorHandler(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		for _, ginErr := range c.Errors {
			logger.Error("Unhandled HTTP Error", zap.Error(ginErr))
		}
	}
}

const defaultTemplate = "index"

func (a *App) createRouter() (*gin.Engine, error) {
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

	gin.SetMode(a.config.RunMode)

	engine := gin.New()
	engine.SetHTMLTemplate(tmplProfiles)
	engine.Use(apiErrorHandler(a.log), gin.Recovery())
	engine.GET("/bans", a.handleGetBans())
	engine.GET("/summary", a.handleGetSummary())
	engine.GET("/profile", a.handleGetProfile())
	engine.GET("/comp", a.handleGetComp())
	engine.GET("/friends", a.handleGetFriendList())
	engine.GET("/sourcebans", a.handleGetSourceBansMany())
	engine.GET("/sourcebans/:steam_id", a.handleGetSourceBans())

	return engine, nil
}

func (a *App) startAPI(ctx context.Context, addr string) error {
	const (
		apiHandlerTimeout = 10 * time.Second
		shutdownTimeout   = 10 * time.Second
	)

	log := a.log.Named("api")

	defer log.Info("Service status changed", zap.String("state", "stopped"))

	httpServer := &http.Server{ //nolint:exhaustruct
		Addr:         addr,
		Handler:      a.router,
		ReadTimeout:  apiHandlerTimeout,
		WriteTimeout: apiHandlerTimeout,
	}

	log.Info("Service status changed", zap.String("state", "ready"))

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if errShutdown := httpServer.Shutdown(shutdownCtx); errShutdown != nil { //nolint:contextcheck
			log.Error("Error shutting down http service", zap.Error(errShutdown))
		}
	}()

	return errors.Wrap(httpServer.ListenAndServe(), "Error returned from HTTP server")
}
