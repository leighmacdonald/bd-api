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
	errTooMany            = errors.New("Too many results requested")
	errInvalidQueryParams = errors.New("Invalid query parameters")
	errInvalidSteamID     = errors.New("Invalid steamid")
	errLoadFailed         = errors.New("Could not load remote resource")
	errInternalError      = errors.New("Internal server error, please try again later")
)

type apiErr struct {
	Error string `json:"error"`
}

func getSteamIDs(w http.ResponseWriter, r *http.Request) (steamid.Collection, bool) {
	steamIDQuery := r.URL.Query().Get("steamids")

	if steamIDQuery == "" {
		responseErr(w, r, http.StatusBadRequest, errInvalidQueryParams, "")

		return nil, false
	}

	var validIDs steamid.Collection

	for _, steamID := range strings.Split(steamIDQuery, ",") {
		sid64 := steamid.New(steamID)

		if !sid64.Valid() {
			responseErr(w, r, http.StatusBadRequest, errInvalidSteamID, "")

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
		responseErr(w, r, http.StatusBadRequest, errTooMany, "")

		return nil, false
	}

	return validIDs, true
}

func handleGetFriendList(cache cache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ids, ok := getSteamIDs(w, r)

		if !ok {
			return
		}

		friends := getSteamFriends(r.Context(), cache, ids)
		responseOk(w, r, http.StatusOK, friends, &baseTmplArgs{ //nolint:exhaustruct
			Title: "Steam Summaries",
		})
	}
}

func handleGetComp(cache cache) http.HandlerFunc {
	log := slog.With(slog.String("fn", runtime.FuncForPC(make([]uintptr, funcSize)[0]).Name()))

	return func(w http.ResponseWriter, r *http.Request) {
		ids, ok := getSteamIDs(w, r)

		if !ok {
			return
		}

		compHistory := getCompHistory(r.Context(), cache, ids)

		if len(ids) != len(compHistory) {
			log.Warn("Failed to fully fetch comp history")
			responseErr(w, r, http.StatusInternalServerError, errLoadFailed, "")

			return
		}
		responseOk(w, r, http.StatusOK, compHistory, &baseTmplArgs{ //nolint:exhaustruct
			Title: "Comp History",
		})
	}
}

func handleGetSummary(cache cache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ids, ok := getSteamIDs(w, r)
		if !ok {
			return
		}

		summaries, errSum := getSteamSummaries(r.Context(), cache, ids)
		if errSum != nil || len(ids) != len(summaries) {
			responseErr(w, r, http.StatusBadRequest, errLoadFailed, "steam api fetch failed")

			return
		}

		responseOk(w, r, http.StatusOK, summaries, &baseTmplArgs{ //nolint:exhaustruct
			Title: "Steam Summaries",
		})
	}
}

func handleGetBans() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ids, ok := getSteamIDs(w, r)

		if !ok {
			return
		}

		bans, errBans := steamweb.GetPlayerBans(r.Context(), ids)

		if errBans != nil || len(ids) != len(bans) {
			responseErr(w, r, http.StatusInternalServerError, errLoadFailed, "")

			return
		}

		responseOk(w, r, http.StatusOK, bans, &baseTmplArgs{ //nolint:exhaustruct
			Title: "Steam Bans",
		})
	}
}

func handleGetProfile(database *pgStore, cache cache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ids, ok := getSteamIDs(w, r)

		if !ok {
			return
		}

		profiles, errProfile := loadProfiles(r.Context(), database, cache, ids)

		if errProfile != nil || len(profiles) == 0 {
			responseErr(w, r, http.StatusInternalServerError, errLoadFailed, "")

			return
		}
		responseOk(w, r, http.StatusOK, profiles, &baseTmplArgs{ //nolint:exhaustruct
			Title: "Profiles",
		})
	}
}

func handleGetSourceBansMany(database *pgStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ids, ok := getSteamIDs(w, r)

		if !ok {
			return
		}

		bans, errBans := database.sbGetBansBySID(r.Context(), ids)
		if errBans != nil {
			responseErr(w, r, http.StatusInternalServerError, errInternalError, "")

			return
		}
		responseOk(w, r, http.StatusOK, bans, &baseTmplArgs{ //nolint:exhaustruct
			Title: "Source Bans",
		})
	}
}

func handleGetSourceBans(database *pgStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sid, ok := steamIDFromSlug(w, r)
		if !ok {
			return
		}

		bans, errBans := database.sbGetBansBySID(r.Context(), steamid.Collection{sid})
		if errBans != nil {
			responseErr(w, r, http.StatusInternalServerError, errInternalError, "")

			return
		}

		out, found := bans[sid.String()]

		if !found || out == nil {
			// Return empty list instead of null
			out = []SbBanRecord{}
		}

		responseOk(w, r, http.StatusOK, out, &baseTmplArgs{ //nolint:exhaustruct
			Title: "Source Bans",
		})
	}
}

func getAttrs(r *http.Request) ([]string, bool) {
	steamIDQuery := r.URL.Query().Get("attrs")
	if steamIDQuery == "" {
		return []string{"cheater"}, true
	}

	attrs := normalizeAttrs(strings.Split(steamIDQuery, ","))
	if len(attrs) == 0 {
		return nil, false
	}

	return attrs, true
}

func handleGetBotDetector(database *pgStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sid, sidOk := getSteamIDs(w, r)
		if !sidOk {
			return
		}

		attrs, attrOk := getAttrs(r)
		if !attrOk {
			return
		}

		results, errSearch := database.bdListSearch(r.Context(), sid, attrs)
		if errSearch != nil {
			responseErr(w, r, http.StatusInternalServerError, errSearch, "internal error")

			return
		}

		if results == nil {
			results = []BDSearchResult{}
		}

		responseOk(w, r, http.StatusOK, results, &baseTmplArgs{ //nolint:exhaustruct
			Title: "TF2BD Search Results",
		})
	}
}
