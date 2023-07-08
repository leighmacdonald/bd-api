package main

import (
	"net/http"
	"runtime"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/bd-api/models"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const funcSize = 10

var (
	ErrTooMany            = errors.New("Too many results requested")
	ErrInvalidQueryParams = errors.New("Invalid query parameters")
	ErrInvalidSteamID     = errors.New("Invalid steamid")
	ErrLoadFailed         = errors.New("Could not load remote resource")
	ErrInternalError      = errors.New("Internal server error, please try again later")
)

type apiErr struct {
	Error string `json:"error"`
}

func newAPIErr(err error) apiErr {
	return apiErr{Error: err.Error()}
}

func getSteamIDS(ctx *gin.Context) (steamid.Collection, bool) {
	steamIDQuery, ok := ctx.GetQuery("steamids")
	if !ok {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, newAPIErr(ErrInvalidQueryParams))

		return nil, false
	}

	var validIds steamid.Collection

	for _, steamID := range strings.Split(steamIDQuery, ",") {
		sid64 := steamid.New(steamID)
		if !sid64.Valid() {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, newAPIErr(ErrInvalidSteamID))

			return nil, false
		}

		validIds = append(validIds, sid64)
	}

	if len(validIds) > maxResults {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, newAPIErr(ErrTooMany))

		return nil, false
	}

	return validIds, true
}

func (a *App) handleGetFriendList() gin.HandlerFunc {
	encoder := newStyleEncoder()

	return func(ctx *gin.Context) {
		ids, ok := getSteamIDS(ctx)
		if !ok {
			return
		}

		renderSyntax(ctx, encoder, a.getSteamFriends(ctx, ids), "profiles", &baseTmplArgs{ //nolint:exhaustruct
			Title: "Steam Summaries",
		})
	}
}

func (a *App) handleGetSummary() gin.HandlerFunc {
	log := a.log.Named(runtime.FuncForPC(make([]uintptr, funcSize)[0]).Name())
	encoder := newStyleEncoder()

	return func(ctx *gin.Context) {
		ids, ok := getSteamIDS(ctx)
		if !ok {
			return
		}

		summaries, errSum := a.getSteamSummaries(ctx, ids)

		if errSum != nil || len(ids) != len(summaries) {
			log.Error("Failed to fetch summary", zap.Error(errSum))
			ctx.AbortWithStatusJSON(http.StatusBadRequest, ErrLoadFailed)

			return
		}

		renderSyntax(ctx, encoder, summaries, "profiles", &baseTmplArgs{ //nolint:exhaustruct
			Title: "Steam Summaries",
		})
	}
}

func (a *App) handleGetBans() gin.HandlerFunc {
	log := a.log.Named(runtime.FuncForPC(make([]uintptr, funcSize)[0]).Name())
	encoder := newStyleEncoder()

	return func(ctx *gin.Context) {
		ids, ok := getSteamIDS(ctx)
		if !ok {
			return
		}

		bans, errBans := steamweb.GetPlayerBans(ctx, ids)
		if errBans != nil || len(ids) != len(bans) {
			log.Error("Failed to fetch player bans", zap.Error(errBans))
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, ErrLoadFailed)

			return
		}

		renderSyntax(ctx, encoder, bans, "profiles", &baseTmplArgs{ //nolint:exhaustruct
			Title: "Steam Bans",
		})
	}
}

func (a *App) handleGetProfile() gin.HandlerFunc {
	log := a.log.Named(runtime.FuncForPC(make([]uintptr, funcSize)[0]).Name())
	encoder := newStyleEncoder()

	return func(ctx *gin.Context) {
		ids, ok := getSteamIDS(ctx)
		if !ok {
			return
		}

		profiles, errProfile := a.loadProfiles(ctx, ids)
		if errProfile != nil || len(profiles) == 0 {
			log.Error("Failed to load profile", zap.Error(errProfile))
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, ErrLoadFailed)

			return
		}

		renderSyntax(ctx, encoder, profiles, "profiles", &baseTmplArgs{ //nolint:exhaustruct
			Title: "Profiles",
		})
	}
}

func (a *App) handleGetSourceBans() gin.HandlerFunc {
	encoder := newStyleEncoder()

	return func(ctx *gin.Context) {
		sid, ok := steamIDFromSlug(ctx)
		if !ok {
			return
		}

		bans, errBans := a.db.sbGetBansBySID(ctx, sid)
		if errBans != nil {
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, ErrInternalError)

			return
		}

		if bans == nil {
			// Return empty list instead of null
			bans = []models.SbBanRecord{}
		}

		renderSyntax(ctx, encoder, bans, "profiles", &baseTmplArgs{ //nolint:exhaustruct
			Title: "source bans",
		})
	}
}
