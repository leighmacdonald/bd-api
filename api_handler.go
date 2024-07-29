package main

import (
	_ "embed"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

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

		var bans []domain.PlayerBanState

		swBans, errBans := steamweb.GetPlayerBans(request.Context(), ids)
		if errBans != nil || len(ids) != len(swBans) {
			responseErr(writer, request, http.StatusInternalServerError, errLoadFailed, "")

			return
		}

		for _, ban := range swBans {
			bans = append(bans, domain.PlayerBanState{
				SteamID:          ban.SteamID,
				CommunityBanned:  ban.CommunityBanned,
				VACBanned:        ban.VACBanned,
				NumberOfVACBans:  ban.NumberOfVACBans,
				DaysSinceLastBan: ban.DaysSinceLastBan,
				NumberOfGameBans: ban.NumberOfGameBans,
				EconomyBan:       ban.EconomyBan,
			})
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

		bans, errBans := database.sourcebansRecordBySID(request.Context(), ids)
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

		bans, errBans := database.sourcebansRecordBySID(request.Context(), steamid.Collection{sid})
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

		results, errSearch := database.botDetectorListSearch(request.Context(), sid, attrs)
		if errSearch != nil {
			responseErr(writer, request, http.StatusInternalServerError, errSearch, "internal error")

			return
		}

		if results == nil {
			results = []domain.BDSearchResult{}
		}

		responseOk(writer, request, results, "TF2BD Search Results")
	}
}

// handleGetLogByID returns an overview for a single logstf match similar to the main logs.tf site. Some info
// is currently omitted such as specific player weapon stats, chatlogs and kill streaks.
func handleGetLogByID(database *pgStore) http.HandlerFunc {
	return func(writer http.ResponseWriter, reader *http.Request) {
		logID, ok := intParam(writer, reader, "log_id")
		if !ok {
			return
		}

		match, errMatch := database.logsTFMatchGet(reader.Context(), logID)
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

		averages, err := database.logsTFPlayerSummary(request.Context(), steamID)
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

		logs, err := database.logsTFMatchList(request.Context(), steamID)
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
		list, err := database.servemeRecords(request.Context())
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

func handleGetSteamID() http.HandlerFunc {
	type sid struct {
		Steam64 string `json:"steam64"`
		Steam32 uint32 `json:"steam32"`
		Steam3  string `json:"steam3"`
		Steam   string `json:"steam"`
		Profile string `json:"profile"`
	}

	return func(writer http.ResponseWriter, request *http.Request) {
		value := request.PathValue("steam_id")
		if value == "" {
			responseErr(writer, request, http.StatusBadRequest, errInvalidSteamID, "Invalid parameter")

			return
		}

		resolved, err := steamid.Resolve(request.Context(), value)
		if err != nil {
			responseErr(writer, request, http.StatusBadRequest, errInvalidSteamID, "Invalid steamid / profile")

			return
		}

		responseOk(writer, request, sid{
			Steam64: resolved.String(),
			Steam32: uint32(resolved.AccountID),
			Steam3:  string(resolved.Steam3()),
			Steam:   string(resolved.Steam(true)),
			Profile: "https://steamcommunity.com/profiles/" + resolved.String(),
		}, fmt.Sprintf("SteamID Conversion (%s)", value))
	}
}

func handleGetStats(database *pgStore) func(http.ResponseWriter, *http.Request) {
	var (
		stats   siteStats
		updated time.Time
		statsMu sync.RWMutex
	)

	return func(writer http.ResponseWriter, request *http.Request) {
		statsMu.RLock()
		timeDiff := time.Since(updated)
		statsMu.RUnlock()

		if timeDiff > time.Minute*15 {
			newStats, errStats := loadStats(request.Context(), database)
			if errStats != nil {
				responseErr(writer, request, http.StatusInternalServerError, errStats, "Failed to generate stats")

				return
			}

			statsMu.Lock()
			updated = time.Now()
			stats = newStats
			statsMu.Unlock()
		}

		statsMu.RLock()
		defer statsMu.RUnlock()

		responseOk(writer, request, stats, "Global Site Stats")
	}
}

func handleGetRGLPlayerHistory(database *pgStore) func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		sids, sidOk := getSteamIDs(writer, request)
		if !sidOk {
			return
		}

		history, err := database.rglPlayerTeamHistory(request.Context(), sids)
		if err != nil {
			responseErr(writer, request, http.StatusInternalServerError, err, "Failed to generate histories")

			return
		}

		histMap := map[string][]domain.RGLPlayerTeamHistory{}

		for _, sid := range sids {
			if _, ok := histMap[sid.String()]; !ok {
				histMap[sid.String()] = []domain.RGLPlayerTeamHistory{}
			}

			for _, hist := range history {
				if sid == hist.SteamID {
					histMap[sid.String()] = append(histMap[sid.String()], hist)
				}
			}
		}

		responseOk(writer, request, histMap, "Player RGL Team Histories")
	}
}

func handleGetRGLList(database *pgStore, config appConfig) func(http.ResponseWriter, *http.Request) {
	//goland:noinspection ALL
	extURL := "http://" + config.ListenAddr + "/"
	if config.ExternalURL != "" {
		extURL = config.ExternalURL
	}

	extURL = strings.TrimSuffix(extURL, "/")

	return func(writer http.ResponseWriter, request *http.Request) {
		bans, errBans := database.rglBansGetAll(request.Context())
		if errBans != nil {
			responseErr(writer, request, http.StatusInternalServerError, errBans, "Failed to get ban list")

			return
		}

		list := domain.TF2BDSchema{
			Schema: "https://raw.githubusercontent.com/leighmacdonald/bd-api/master/schemas/playerlist.schema.json",
			FileInfo: domain.FileInfo{
				Authors:     []string{"rgl league", "bd-api"},
				Description: "All league bans and infractions",
				Title:       "RGL.gg Bans",
				UpdateURL:   extURL + "/list/rgl",
			},
			Players: make([]domain.TF2BDPlayer, len(bans)),
		}

		for banIdx, ban := range bans {
			player := domain.TF2BDPlayer{
				Attributes: []string{"rgl"},
				LastSeen: domain.LastSeen{
					PlayerName: ban.Alias,
					Time:       int(ban.CreatedAt.Unix()),
				},
				Steamid: ban.SteamID,
				Proof:   []string{ban.Reason},
			}
			if ban.ExpiresAt.After(time.Date(2500, 0, 0, 0, 0, 0, 0, time.UTC)) {
				player.Proof = append(player.Proof, "Permanent Ban")
			}

			list.Players[banIdx] = player
		}

		responseOk(writer, request, list, "RGL Ban List")
	}
}

func handleGetServemeListBD(database *pgStore, config appConfig) func(http.ResponseWriter, *http.Request) {
	//goland:noinspection ALL
	extURL := "http://" + config.ListenAddr + "/"
	if config.ExternalURL != "" {
		extURL = config.ExternalURL
	}

	extURL = strings.TrimSuffix(extURL, "/")

	return func(writer http.ResponseWriter, request *http.Request) {
		bans, errBans := database.servemeRecords(request.Context())
		if errBans != nil {
			responseErr(writer, request, http.StatusInternalServerError, errBans, "Failed to get serveme ban list")

			return
		}

		list := domain.TF2BDSchema{
			Schema: "https://raw.githubusercontent.com/leighmacdonald/bd-api/master/schemas/playerlist.schema.json",
			FileInfo: domain.FileInfo{
				Authors:     []string{"serveme.tf", "bd-api"},
				Description: "All serveme.tf bans",
				Title:       "serveme.tf Bans",
				UpdateURL:   extURL + "/list/serveme",
			},
			Players: make([]domain.TF2BDPlayer, len(bans)),
		}

		for banIdx, ban := range bans {
			player := domain.TF2BDPlayer{
				Attributes: []string{"serveme"},
				LastSeen: domain.LastSeen{
					PlayerName: ban.Name,
					Time:       int(ban.CreatedOn.Unix()),
				},
				Steamid: ban.SteamID,
				Proof:   []string{ban.Reason, "Permanent Ban"},
			}

			list.Players[banIdx] = player
		}

		responseOk(writer, request, list, "Serveme Ban List")
	}
}
