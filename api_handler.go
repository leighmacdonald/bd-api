package main

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime"
	"strconv"
	"strings"

	"github.com/leighmacdonald/bd-api/domain"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/steamweb/v2"
)

const funcSize = 10

var (
	errTooMany            = errors.New("too many results requested")
	errInvalidQueryParams = errors.New("invalid query parameters")
	errInvalidSteamID     = errors.New("invalid steamid")
	errLoadFailed         = errors.New("could not load remote resource")
	errInternalError      = errors.New("internal server error, please try again later")
)

type apiErr struct {
	Error string `json:"error"`
}

func getSteamIDs(writer http.ResponseWriter, request *http.Request) (steamid.Collection, bool) {
	steamIDQuery := request.URL.Query().Get("steamids")

	if steamIDQuery == "" {
		responseErr(writer, request, http.StatusBadRequest, errInvalidQueryParams, "")

		return nil, false
	}

	var validIDs steamid.Collection

	for _, steamID := range strings.Split(steamIDQuery, ",") {
		sid64 := steamid.New(steamID)

		if !sid64.Valid() {
			responseErr(writer, request, http.StatusBadRequest, errInvalidSteamID, "")

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
		responseErr(writer, request, http.StatusBadRequest, errTooMany, "")

		return nil, false
	}

	return validIDs, true
}

func handleGetFriendList(cache cache) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		ids, ok := getSteamIDs(writer, request)

		if !ok {
			return
		}

		friends := getSteamFriends(request.Context(), cache, ids)
		responseOk(writer, request, friends, "Steam Summaries")
	}
}

func handleGetComp(cache cache) http.HandlerFunc {
	log := slog.With(slog.String("fn", runtime.FuncForPC(make([]uintptr, funcSize)[0]).Name()))

	return func(writer http.ResponseWriter, request *http.Request) {
		ids, ok := getSteamIDs(writer, request)

		if !ok {
			return
		}

		compHistory := getCompHistory(request.Context(), cache, ids)

		if len(ids) != len(compHistory) {
			log.Warn("Failed to fully fetch comp history")
			responseErr(writer, request, http.StatusInternalServerError, errLoadFailed, "")

			return
		}
		responseOk(writer, request, compHistory, "Comp History")
	}
}

func handleGetSummary(cache cache) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		ids, ok := getSteamIDs(writer, request)
		if !ok {
			return
		}

		summaries, errSum := getSteamSummaries(request.Context(), cache, ids)
		if errSum != nil || len(ids) != len(summaries) {
			responseErr(writer, request, http.StatusBadRequest, errLoadFailed, "steam api fetch failed")

			return
		}

		responseOk(writer, request, summaries, "Steam Summaries")
	}
}

func handleGetBans() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		ids, ok := getSteamIDs(writer, request)

		if !ok {
			return
		}

		bans, errBans := steamweb.GetPlayerBans(request.Context(), ids)

		if errBans != nil || len(ids) != len(bans) {
			responseErr(writer, request, http.StatusInternalServerError, errLoadFailed, "")

			return
		}

		responseOk(writer, request, bans, "Steam Bans")
	}
}

func handleGetProfile(database *pgStore, cache cache) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		ids, ok := getSteamIDs(writer, request)

		if !ok {
			return
		}

		profiles, errProfile := loadProfiles(request.Context(), database, cache, ids)

		if errProfile != nil || len(profiles) == 0 {
			responseErr(writer, request, http.StatusInternalServerError, errLoadFailed, "")

			return
		}

		responseOk(writer, request, profiles, "Profiles")
	}
}

func handleGetSourceBansMany(database *pgStore) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		ids, ok := getSteamIDs(writer, request)

		if !ok {
			return
		}

		bans, errBans := database.sbGetBansBySID(request.Context(), ids)
		if errBans != nil {
			responseErr(writer, request, http.StatusInternalServerError, errInternalError, "")

			return
		}
		responseOk(writer, request, bans, "Source Bans")
	}
}

func handleGetSourceBans(database *pgStore) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		sid, ok := steamIDFromSlug(writer, request)
		if !ok {
			return
		}

		bans, errBans := database.sbGetBansBySID(request.Context(), steamid.Collection{sid})
		if errBans != nil {
			responseErr(writer, request, http.StatusInternalServerError, errInternalError, "")

			return
		}

		out, found := bans[sid.String()]

		if !found || out == nil {
			// Return empty list instead of null
			out = []domain.SbBanRecord{}
		}

		responseOk(writer, request, out, "Source Bans")
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
	return func(writer http.ResponseWriter, request *http.Request) {
		sid, sidOk := getSteamIDs(writer, request)
		if !sidOk {
			return
		}

		attrs, attrOk := getAttrs(request)
		if !attrOk {
			return
		}

		results, errSearch := database.bdListSearch(request.Context(), sid, attrs)
		if errSearch != nil {
			responseErr(writer, request, http.StatusInternalServerError, errSearch, "internal error")

			return
		}

		if results == nil {
			results = []BDSearchResult{}
		}

		responseOk(writer, request, results, "TF2BD Search Results")
	}
}

func intParam(w http.ResponseWriter, r *http.Request, param string) (int, bool) {
	intStr := r.PathValue(param)
	if intStr == "" {
		responseErr(w, r, http.StatusBadRequest, errInvalidSteamID, "Invalid parameter")

		return 0, false
	}

	intVal, err := strconv.Atoi(intStr)
	if err != nil {
		return 0, false
	}

	return intVal, true
}

func handleGetLogByID(database *pgStore) http.HandlerFunc {
	return func(writer http.ResponseWriter, reader *http.Request) {
		logID, ok := intParam(writer, reader, "log_id")
		if !ok {
			return
		}

		match, errMatch := database.getLogsTFMatch(reader.Context(), logID)
		if errMatch != nil {
			if errors.Is(errMatch, errDatabaseNoResults) {
				responseErr(writer, reader, http.StatusNotFound, errDatabaseNoResults, "Unknown match id")

				return
			}

			responseErr(writer, reader, http.StatusInternalServerError, errInternalError, "Unhandled error")

			return
		}

		responseOk(writer, reader, match, fmt.Sprintf("Match Log #%d - %s", match.LogID, match.Title))
	}
}

func handleGetLogsSummary(database *pgStore) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		steamID, found := steamIDFromSlug(writer, request)
		if !found {
			return
		}

		averages, err := database.getLogsTFPlayerSummary(request.Context(), steamID)
		if err != nil {
			if errors.Is(err, errDatabaseNoResults) {
				responseErr(writer, request, http.StatusNotFound, errDatabaseNoResults, "Unknown match id")

				return
			}

			responseErr(writer, request, http.StatusInternalServerError, errInternalError, "Unhandled error")

			return
		}

		responseOk(writer, request, averages, fmt.Sprintf("Logs.tf Summary %s", steamID.String()))
	}
}

func handleGetLogsList(database *pgStore) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		steamID, found := steamIDFromSlug(writer, request)
		if !found {
			return
		}

		logs, err := database.getLogsTFList(request.Context(), steamID)
		if err != nil {
			if errors.Is(err, errDatabaseNoResults) {
				responseErr(writer, request, http.StatusNotFound, errDatabaseNoResults, "Unknown match id")

				return
			}

			responseErr(writer, request, http.StatusInternalServerError, errInternalError, "Unhandled error")

			return
		}

		if logs == nil {
			logs = []domain.LogsTFMatchInfo{}
		}

		responseOk(writer, request, logs, fmt.Sprintf("Logs.tf List %s", steamID.String()))
	}
}
