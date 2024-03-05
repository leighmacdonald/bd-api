package main

import (
	"log/slog"
	"net/http"
	"runtime"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/bd-api/models"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"github.com/pkg/errors"
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

func getSteamIDs(ctx *gin.Context) (steamid.Collection, bool) {
	steamIDQuery, ok := ctx.GetQuery("steamids")
	if !ok {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, newAPIErr(ErrInvalidQueryParams))

		return nil, false
	}

	var validIDs steamid.Collection

	for _, steamID := range strings.Split(steamIDQuery, ",") {
		sid64 := steamid.New(steamID)
		if !sid64.Valid() {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, newAPIErr(ErrInvalidSteamID))

			return nil, false
		}

		unique := true

		for _, knownID := range validIDs {
			if knownID == sid64 {
				unique = false

				break
			}
		}

		if unique {
			validIDs = append(validIDs, sid64)
		}
	}

	if len(validIDs) > maxResults {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, newAPIErr(ErrTooMany))

		return nil, false
	}

	return validIDs, true
}

func (a *App) handleGetFriendList() gin.HandlerFunc {
	encoder := newStyleEncoder()

	return func(ctx *gin.Context) {
		ids, ok := getSteamIDs(ctx)
		if !ok {
			return
		}

		renderSyntax(ctx, encoder, a.getSteamFriends(ctx, ids), defaultTemplate, &baseTmplArgs{ //nolint:exhaustruct
			Title: "Steam Summaries",
		})
	}
}

func (a *App) handleGetComp() gin.HandlerFunc {
	log := slog.With(slog.String("fn", runtime.FuncForPC(make([]uintptr, funcSize)[0]).Name()))
	encoder := newStyleEncoder()

	return func(ctx *gin.Context) {
		ids, ok := getSteamIDs(ctx)
		if !ok {
			return
		}

		compHistory := a.getCompHistory(ctx, ids)

		if len(ids) != len(compHistory) {
			log.Warn("Failed to fully fetch comp history")
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, ErrLoadFailed)

			return
		}

		renderSyntax(ctx, encoder, compHistory, defaultTemplate, &baseTmplArgs{ //nolint:exhaustruct
			Title: "Comp History",
		})
	}
}

func (a *App) handleGetSummary() gin.HandlerFunc {
	log := slog.With(slog.String("fn", runtime.FuncForPC(make([]uintptr, funcSize)[0]).Name()))
	encoder := newStyleEncoder()

	return func(ctx *gin.Context) {
		ids, ok := getSteamIDs(ctx)
		if !ok {
			return
		}

		summaries, errSum := a.getSteamSummaries(ctx, ids)

		if errSum != nil || len(ids) != len(summaries) {
			log.Error("Failed to fetch summary", ErrAttr(errSum))
			ctx.AbortWithStatusJSON(http.StatusBadRequest, ErrLoadFailed)

			return
		}

		renderSyntax(ctx, encoder, summaries, defaultTemplate, &baseTmplArgs{ //nolint:exhaustruct
			Title: "Steam Summaries",
		})
	}
}

func (a *App) handleGetBans() gin.HandlerFunc {
	log := slog.With(slog.String("fn", runtime.FuncForPC(make([]uintptr, funcSize)[0]).Name()))
	encoder := newStyleEncoder()

	return func(ctx *gin.Context) {
		ids, ok := getSteamIDs(ctx)
		if !ok {
			return
		}

		bans, errBans := steamweb.GetPlayerBans(ctx, ids)
		if errBans != nil || len(ids) != len(bans) {
			log.Error("Failed to fetch player bans", ErrAttr(errBans))
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, ErrLoadFailed)

			return
		}

		renderSyntax(ctx, encoder, bans, defaultTemplate, &baseTmplArgs{ //nolint:exhaustruct
			Title: "Steam Bans",
		})
	}
}

func (a *App) handleGetProfile() gin.HandlerFunc {
	log := slog.With(slog.String("fn", runtime.FuncForPC(make([]uintptr, funcSize)[0]).Name()))
	encoder := newStyleEncoder()

	return func(ctx *gin.Context) {
		ids, ok := getSteamIDs(ctx)
		if !ok {
			return
		}

		profiles, errProfile := a.loadProfiles(ctx, ids)
		if errProfile != nil || len(profiles) == 0 {
			log.Error("Failed to load profile", ErrAttr(errProfile))
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, ErrLoadFailed)

			return
		}

		renderSyntax(ctx, encoder, profiles, defaultTemplate, &baseTmplArgs{ //nolint:exhaustruct
			Title: "Profiles",
		})
	}
}

func (a *App) handleGetSourceBansMany() gin.HandlerFunc {
	log := slog.With(slog.String("fn", runtime.FuncForPC(make([]uintptr, funcSize)[0]).Name()))
	encoder := newStyleEncoder()

	return func(ctx *gin.Context) {
		ids, ok := getSteamIDs(ctx)
		if !ok {
			return
		}

		bans, errBans := a.db.sbGetBansBySID(ctx, ids)
		if errBans != nil {
			log.Error("Failed to query bans from database", ErrAttr(errBans))
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, ErrInternalError)

			return
		}

		renderSyntax(ctx, encoder, bans, defaultTemplate, &baseTmplArgs{ //nolint:exhaustruct
			Title: "Source Bans",
		})
	}
}

func (a *App) handleGetSourceBans() gin.HandlerFunc {
	log := slog.With(slog.String("fn", runtime.FuncForPC(make([]uintptr, funcSize)[0]).Name()))
	encoder := newStyleEncoder()

	return func(ctx *gin.Context) {
		sid, ok := steamIDFromSlug(ctx)
		if !ok {
			return
		}

		bans, errBans := a.db.sbGetBansBySID(ctx, steamid.Collection{sid})
		if errBans != nil {
			log.Error("Failed to query bans from database", ErrAttr(errBans))
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, ErrInternalError)

			return
		}

		out, found := bans[sid]
		if !found || out == nil {
			// Return empty list instead of null
			out = []models.SbBanRecord{}
		}

		renderSyntax(ctx, encoder, out, defaultTemplate, &baseTmplArgs{ //nolint:exhaustruct
			Title: "Source Bans",
		})
	}
}
