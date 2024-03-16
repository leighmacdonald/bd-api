package main

import (
	"log/slog"
	"net/http"
	"runtime"
	"strings"

	"github.com/leighmacdonald/steamid/v4/steamid"
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

func getSteamIDs(w http.ResponseWriter, r *http.Request) (steamid.Collection, bool) {
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

func handleGetFriendList(cache cache) http.HandlerFunc {
	encoder := newStyleEncoder()

	return func(w http.ResponseWriter, r *http.Request) {
		ids, ok := getSteamIDs(ctx)

		if !ok {
			return
		}

		renderResponse(ctx, encoder, getSteamFriends(ctx, cache, ids), &baseTmplArgs{ //nolint:exhaustruct
			Title: "Steam Summaries",
		})
	}
}

func handleGetComp(cache cache) http.HandlerFunc {
	log := slog.With(slog.String("fn", runtime.FuncForPC(make([]uintptr, funcSize)[0]).Name()))

	encoder := newStyleEncoder()

	return func(w http.ResponseWriter, r *http.Request) {
		ids, ok := getSteamIDs(ctx)

		if !ok {
			return
		}

		compHistory := getCompHistory(ctx, cache, ids)

		if len(ids) != len(compHistory) {
			log.Warn("Failed to fully fetch comp history")
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, ErrLoadFailed)

			return
		}

		renderResponse(ctx, encoder, compHistory, &baseTmplArgs{ //nolint:exhaustruct
			Title: "Comp History",
		})
	}
}

func handleGetSummary(cache cache) http.HandlerFunc {
	log := slog.With(slog.String("fn", runtime.FuncForPC(make([]uintptr, funcSize)[0]).Name()))

	encoder := newStyleEncoder()

	return func(w http.ResponseWriter, r *http.Request) {
		ids, ok := getSteamIDs(ctx)

		if !ok {
			return
		}

		summaries, errSum := getSteamSummaries(ctx, cache, ids)

		if errSum != nil || len(ids) != len(summaries) {
			log.Error("Failed to fetch summary", ErrAttr(errSum))
			ctx.AbortWithStatusJSON(http.StatusBadRequest, ErrLoadFailed)

			return
		}

		renderResponse(ctx, encoder, summaries, &baseTmplArgs{ //nolint:exhaustruct
			Title: "Steam Summaries",
		})
	}
}

func handleGetBans() http.HandlerFunc {
	log := slog.With(slog.String("fn", runtime.FuncForPC(make([]uintptr, funcSize)[0]).Name()))

	encoder := newStyleEncoder()

	return func(w http.ResponseWriter, r *http.Request) {
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

		renderResponse(ctx, encoder, bans, &baseTmplArgs{ //nolint:exhaustruct
			Title: "Steam Bans",
		})
	}
}

func handleGetProfile(database *pgStore, cache cache) http.HandlerFunc {
	log := slog.With(slog.String("fn", runtime.FuncForPC(make([]uintptr, funcSize)[0]).Name()))

	encoder := newStyleEncoder()

	return func(w http.ResponseWriter, r *http.Request) {
		ids, ok := getSteamIDs(ctx)

		if !ok {
			return
		}

		profiles, errProfile := loadProfiles(ctx, database, cache, ids)

		if errProfile != nil || len(profiles) == 0 {
			log.Error("Failed to load profile", ErrAttr(errProfile))
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, ErrLoadFailed)

			return
		}

		renderResponse(ctx, encoder, profiles, &baseTmplArgs{ //nolint:exhaustruct
			Title: "Profiles",
		})
	}
}

func handleGetSourceBansMany(database *pgStore) http.HandlerFunc {
	log := slog.With(slog.String("fn", runtime.FuncForPC(make([]uintptr, funcSize)[0]).Name()))

	encoder := newStyleEncoder()

	return func(w http.ResponseWriter, r *http.Request) {
		ids, ok := getSteamIDs(ctx)

		if !ok {
			return
		}

		bans, errBans := database.sbGetBansBySID(ctx, ids)
		if errBans != nil {
			log.Error("Failed to query bans from database", ErrAttr(errBans))
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, ErrInternalError)

			return
		}

		renderResponse(ctx, encoder, bans, &baseTmplArgs{ //nolint:exhaustruct
			Title: "Source Bans",
		})
	}
}

func handleGetSourceBans(database *pgStore) http.HandlerFunc {
	encoder := newStyleEncoder()

	return func(w http.ResponseWriter, r *http.Request) {
		sid, ok := steamIDFromSlug(r)
		if !ok {
			return
		}

		bans, errBans := database.sbGetBansBySID(ctx, steamid.Collection{sid})
		if errBans != nil {
			slog.Error("Failed to query bans from database", ErrAttr(errBans))
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, ErrInternalError)

			return
		}

		out, found := bans[sid.String()]

		if !found || out == nil {
			// Return empty list instead of null
			out = []SbBanRecord{}
		}

		renderResponse(ctx, encoder, out, &baseTmplArgs{ //nolint:exhaustruct
			Title: "Source Bans",
		})
	}
}

func getAttrs(w http.ResponseWriter, r *http.Request) ([]string, bool) {
	steamIDQuery, ok := ctx.GetQuery("attrs")
	if !ok {
		return []string{"cheater"}, true
	}

	attrs := normalizeAttrs(strings.Split(steamIDQuery, ","))
	if len(attrs) == 0 {
		return nil, false
	}

	return attrs, true
}

func handleGetBotDetector(database *pgStore) http.HandlerFunc {
	encoder := newStyleEncoder()

	return func(w http.ResponseWriter, r *http.Request) {
		sid, sidOk := getSteamIDs(r)
		if !sidOk {
			return
		}

		attrs, attrOk := getAttrs(r)
		if !attrOk {
			return
		}

		results, errSearch := database.bdListSearch(ctx, sid, attrs)
		if errSearch != nil {
			slog.Error("Failed to query bd lists from database", ErrAttr(errSearch))
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, ErrInternalError)

			return
		}

		if results == nil {
			results = []BDSearchResult{}
		}

		renderResponse(ctx, encoder, results, &baseTmplArgs{ //nolint:exhaustruct
			Title: "TF2BD Search Results",
		})
	}
}
