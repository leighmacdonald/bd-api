package main

import (
	_ "embed"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"

	"github.com/leighmacdonald/bd-api/domain"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/steamweb/v2"
)

// handleGetFriendList returns a list of the users friends. If the users friends are private,
// no results are returned.
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

// handleGetComp returns a list of a users competitive history.
// This is very incomplete currently.
func handleGetComp(cache cache) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		ids, ok := getSteamIDs(writer, request)

		if !ok {
			return
		}

		compHistory := getCompHistory(request.Context(), cache, ids)

		if len(ids) != len(compHistory) {
			slog.Warn("Failed to fully fetch comp history")
			responseErr(writer, request, http.StatusInternalServerError, errLoadFailed, "")

			return
		}
		responseOk(writer, request, compHistory, "Comp History")
	}
}

// handleGetSummary returns a players steam profile summary. This mirrors the data shape in the steam summary api.
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

// handleGetBans returns the ban state of the player from the steam api.
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

// handleGetProfile returns a composite of all known data on the players.
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

// handleGetSourceBansMany fetches the indexed sourcebans data for multiple users.
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

// handleGetSourceBans fetches a single users sourcebans data.
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

// handleGetBotDetector searches the tracked bot detector lists for matches. Supports multiple steamids.
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

// handleGetLogByID returns a overview for a single logstf match similar to the main logs.tf site. Some info
// is currently omitted such as specific player weapon stats, chatlogs and kill streaks.
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

// handleGetLogsSummary returns a summary of a players logstf match statistics.
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

// handleGetLogsList returns a list of a users logstf matches.
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

// handleGetServemeList returns a list of all known serveme bans.
func handleGetServemeList(database *pgStore) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		list, err := database.getServeMeList(request.Context())
		if err != nil && !errors.Is(err, errDatabaseNoResults) {
			responseErr(writer, request, http.StatusInternalServerError, errInternalError, "Unhandled error")

			return
		}

		if list == nil {
			list = []domain.ServeMeRecord{}
		}

		responseOk(writer, request, list, fmt.Sprintf("Serveme Ban Records (%d)", len(list)))
	}
}

type tmplVars struct {
	Version string
}

//go:embed index.tmpl.html
var tmplLoginPage string

func handleGetIndex() http.HandlerFunc {
	index, err := template.New("index").Parse(tmplLoginPage)
	if err != nil {
		panic(err)
	}

	return func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "text/html")
		writer.WriteHeader(http.StatusOK)
		if errTmpl := index.Execute(writer, tmplVars{Version: version}); errTmpl != nil {
			slog.Error("Failed to execute index template", ErrAttr(errTmpl))
		}
	}
}
