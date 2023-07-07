package main

import (
	"net/http"
	"runtime"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/bd-api/models"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"go.uber.org/zap"
)

const funcSize = 10

func getSteamIDS(ctx *gin.Context) (steamid.Collection, bool) {
	steamIDQuery, ok := ctx.GetQuery("steamids")
	if !ok {
		ctx.AbortWithStatus(http.StatusBadRequest)

		return nil, false
	}

	var validIds steamid.Collection

	for _, steamID := range strings.Split(steamIDQuery, ",") {
		sid64 := steamid.New(steamID)
		if !sid64.Valid() {
			ctx.AbortWithStatus(http.StatusBadRequest)

			return nil, false
		}

		validIds = append(validIds, sid64)
	}

	return validIds, true
}

func (a *App) handleGetFriendList() gin.HandlerFunc {
	log := a.log.Named(runtime.FuncForPC(make([]uintptr, funcSize)[0]).Name())

	return func(ctx *gin.Context) {
		ids, ok := getSteamIDS(ctx)
		if !ok {
			return
		}

		summaries, errSum := a.getSteamSummaries(ctx, ids)

		if errSum != nil || len(ids) != len(summaries) {
			log.Error("Failed to fetch summary", zap.Error(errSum))
			ctx.AbortWithStatus(http.StatusBadRequest)

			return
		}

		ctx.JSON(http.StatusOK, summaries)
	}
}

func (a *App) handleGetSummary() gin.HandlerFunc {
	log := a.log.Named(runtime.FuncForPC(make([]uintptr, funcSize)[0]).Name())

	return func(ctx *gin.Context) {
		ids, ok := getSteamIDS(ctx)
		if !ok {
			return
		}

		summaries, errSum := a.getSteamSummaries(ctx, ids)

		if errSum != nil || len(ids) != len(summaries) {
			log.Error("Failed to fetch summary", zap.Error(errSum))
			ctx.AbortWithStatus(http.StatusBadRequest)

			return
		}

		ctx.JSON(http.StatusOK, summaries)
	}
}

func (a *App) handleGetBans() gin.HandlerFunc {
	log := a.log.Named(runtime.FuncForPC(make([]uintptr, funcSize)[0]).Name())

	return func(ctx *gin.Context) {
		ids, ok := getSteamIDS(ctx)
		if !ok {
			return
		}

		bans, errBans := steamweb.GetPlayerBans(ctx, ids)
		if errBans != nil || len(ids) != len(bans) {
			log.Error("Failed to fetch player bans", zap.Error(errBans))
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
		ids, ok := getSteamIDS(ctx)
		if !ok {
			return
		}

		profiles, errProfile := a.loadProfiles(ctx, ids)
		if errProfile != nil || len(profiles) == 0 {
			log.Error("Failed to load profile", zap.Error(errProfile))
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, "Failed to fetch profile")

			return
		}

		ctx.JSON(http.StatusOK, profiles)
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
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, "Failed to query records")

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
