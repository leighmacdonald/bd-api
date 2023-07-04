package main

import (
	"net/http"
	"runtime"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"go.uber.org/zap"
)

const funcSize = 10

func (a *App) handleGetSummary() gin.HandlerFunc {
	log := a.log.Named(runtime.FuncForPC(make([]uintptr, funcSize)[0]).Name())

	return func(ctx *gin.Context) {
		steamIDQuery, ok := ctx.GetQuery("steam_id")
		if !ok {
			ctx.AbortWithStatus(http.StatusBadRequest)

			return
		}

		sid64, steamIDErr := steamid.SID64FromString(steamIDQuery)
		if steamIDErr != nil {
			ctx.AbortWithStatus(http.StatusBadRequest)

			return
		}

		if !sid64.Valid() {
			ctx.AbortWithStatus(http.StatusBadRequest)

			return
		}

		ids := steamid.Collection{sid64}
		summaries, errSum := a.getSteamSummary(ctx, sid64)

		if errSum != nil || len(ids) != len(summaries) {
			log.Error("Failed to fetch summary",
				zap.Error(errSum), zap.Int64("steam_id", sid64.Int64()))
			ctx.AbortWithStatus(http.StatusBadRequest)

			return
		}

		ctx.JSON(http.StatusOK, summaries)
	}
}

func (a *App) handleGetBans() gin.HandlerFunc {
	log := a.log.Named(runtime.FuncForPC(make([]uintptr, funcSize)[0]).Name())

	return func(ctx *gin.Context) {
		steamIDQuery, ok := ctx.GetQuery("steam_id")
		if !ok {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, "Missing steam_id")

			return
		}

		sid, steamIDErr := steamid.SID64FromString(steamIDQuery)
		if steamIDErr != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, "Invalid steam_id")

			return
		}

		ids := steamid.Collection{sid}

		bans, errBans := steamweb.GetPlayerBans(ctx, ids)
		if errBans != nil || len(ids) != len(bans) {
			log.Error("Failed to fetch player bans",
				zap.Error(errBans), zap.Int64("steam_id", sid.Int64()))
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, "Failed to fetch bans")

			return
		}

		if bans == nil {
			// Return empty list instead of null
			bans = []steamweb.PlayerBanState{}
		}

		ctx.JSON(http.StatusOK, bans)
	}
}

func (a *App) handleGetProfile() gin.HandlerFunc {
	log := a.log.Named(runtime.FuncForPC(make([]uintptr, funcSize)[0]).Name())

	return func(ctx *gin.Context) {
		steamIDQuery, ok := ctx.GetQuery("steam_id")
		if !ok {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, "Missing steam_id")

			return
		}

		sid64, steamIDErr := steamid.SID64FromString(steamIDQuery)
		if steamIDErr != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, "Invalid steam_id")

			return
		}

		var profile Profile
		if errProfile := a.loadProfile(ctx, log, sid64, &profile); errProfile != nil {
			log.Error("Failed to load profile", zap.Error(errProfile))
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, "Failed to fetch profile")

			return
		}

		ctx.JSON(http.StatusOK, profile)
	}
}

func (a *App) handleGetSourceBans() gin.HandlerFunc {
	encoder := newStyleEncoder()

	return func(ctx *gin.Context) {
		sid, errSid := steamIDFromSlug(ctx)
		if errSid != nil {
			return
		}

		bans, errBans := a.db.sbGetBansBySID(ctx, sid)
		if errBans != nil {
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, "Failed to query records")

			return
		}

		if bans == nil {
			// Return empty list instead of null
			bans = []sbBanRecord{}
		}

		renderSyntax(ctx, encoder, bans, "profiles", &baseTmplArgs{ //nolint:exhaustruct
			Title: "source bans",
		})
	}
}

func (a *App) handleGetProfiles() gin.HandlerFunc {
	log := a.log.Named(runtime.FuncForPC(make([]uintptr, funcSize)[0]).Name())
	encoder := newStyleEncoder()

	return func(ctx *gin.Context) {
		sid, errSid := steamIDFromSlug(ctx)
		if errSid != nil {
			log.Error("Failed to resolve slug steamid", zap.Error(errSid))

			return
		}

		var profile Profile
		if errProfile := a.loadProfile(ctx, log, sid, &profile); errProfile != nil {
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, "Failed to load profile")

			return
		}

		renderSyntax(ctx, encoder, profile, "profiles", &baseTmplArgs{ //nolint:exhaustruct
			Title: profile.Summary.PersonaName,
		})
	}
}
