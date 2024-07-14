package main

import (
	"context"
	"errors"
	"html/template"
	"log/slog"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/leighmacdonald/bd-api/domain"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/steamweb/v2"
)

const (
	maxResults      = 100
	shutdownTimeout = 10 * time.Second
	apiTimeout      = time.Second * 10
)

var (
	indexTMPL             *template.Template
	encoder               *styleEncoder
	errParseTemplate      = errors.New("failed to parse html template")
	errInvalidSteamID     = errors.New("invalid steamid")
	errInvalidQueryParams = errors.New("invalid query parameters")
	errTooMany            = errors.New("too many results requested")
	errLoadFailed         = errors.New("could not load remote resource")
	errInternalError      = errors.New("internal server error, please try again later")
)

// loadProfiles concurrently loads data from all of the tracked data source tables and assembles them into
// a slice of domain.Profile.
func loadProfiles(ctx context.Context, database *pgStore, cache cache, steamIDs steamid.Collection) ([]domain.Profile, error) { //nolint:funlen
	var ( //nolint:prealloc
		waitGroup   = &sync.WaitGroup{}
		summaries   []steamweb.PlayerSummary
		bans        []steamweb.PlayerBanState
		profiles    []domain.Profile
		logs        map[steamid.SteamID]int
		friends     friendMap
		servemeBans []*domain.ServeMeRecord
		sourceBans  BanRecordMap
	)

	if len(steamIDs) > maxResults {
		return nil, errTooMany
	}

	localCtx, cancel := context.WithTimeout(ctx, apiTimeout)
	defer cancel()

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		sbRecords, errSB := database.sbGetBansBySID(localCtx, steamIDs)
		if errSB != nil {
			slog.Error("Failed to load sourcebans records", ErrAttr(errSB))
		}

		sourceBans = sbRecords
	}()

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		sum, errSum := getSteamSummaries(localCtx, cache, steamIDs)
		if errSum != nil || len(sum) == 0 {
			slog.Error("Failed to load player summaries", ErrAttr(errSum))
		}

		summaries = sum
	}()

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		banState, errBanState := getSteamBans(localCtx, cache, steamIDs)
		if errBanState != nil || len(banState) == 0 {
			slog.Error("Failed to load player ban states", ErrAttr(errBanState))
		}

		bans = banState
	}()

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		friends = getSteamFriends(localCtx, cache, steamIDs)
	}()

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		serveme, errs := database.getServeMeRecords(localCtx, steamIDs)
		if errs != nil && !errors.Is(errs, errDatabaseNoResults) {
			slog.Error("Failed to get serveme records")

			return
		}

		servemeBans = serveme
	}()

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		logsVal, err := database.getLogsTFCount(localCtx, steamIDs)
		if err != nil {
			slog.Error("failed to query log counts", ErrAttr(err))

			return
		}

		logs = logsVal
	}()

	waitGroup.Wait()

	if len(steamIDs) == 0 || len(summaries) == 0 {
		return nil, errDatabaseNoResults
	}

	for _, sid := range steamIDs {
		var profile domain.Profile

		for _, summary := range summaries {
			if summary.SteamID == sid {
				profile.Summary = summary

				break
			}
		}

		for _, ban := range bans {
			if ban.SteamID == sid {
				profile.BanState = domain.PlayerBanState{
					SteamID:          ban.SteamID,
					CommunityBanned:  ban.CommunityBanned,
					VACBanned:        ban.VACBanned,
					NumberOfVACBans:  ban.NumberOfVACBans,
					DaysSinceLastBan: ban.DaysSinceLastBan,
					NumberOfGameBans: ban.NumberOfGameBans,
					EconomyBan:       ban.EconomyBan,
				}

				break
			}
		}

		for _, serveme := range servemeBans {
			if serveme.SteamID == sid {
				profile.ServeMe = serveme

				break
			}
		}

		for logSID, count := range logs {
			if logSID == sid {
				profile.LogsCount = count

				break
			}
		}

		if records, ok := sourceBans[sid.String()]; ok {
			profile.SourceBans = records
		} else {
			// Dont return null json values
			profile.SourceBans = []domain.SbBanRecord{}
		}

		if friendsList, ok := friends[sid.String()]; ok {
			profile.Friends = friendsList
		} else {
			profile.Friends = []steamweb.Friend{}
		}

		profile.Seasons = []domain.Season{}
		sort.Slice(profile.Seasons, func(i, j int) bool {
			return profile.Seasons[i].DivisionInt < profile.Seasons[j].DivisionInt
		})
		profiles = append(profiles, profile)
	}

	return profiles, nil
}

type apiErr struct {
	Error string `json:"error"`
}

func responseErr(w http.ResponseWriter, r *http.Request, status int, err error, userMsg string) {
	msg := err.Error()
	if userMsg != "" {
		msg = userMsg
	}
	renderResponse(w, r, status, map[string]string{"error": msg}, "Error")
	slog.Error("error executing request", ErrAttr(err))
}

func responseOk(w http.ResponseWriter, r *http.Request, data any, title string) {
	renderResponse(w, r, http.StatusOK, data, title)
}

func renderResponse(writer http.ResponseWriter, request *http.Request, status int, data any, title string) {
	if data == nil {
		data = []string{}
	}

	if strings.Contains(strings.ToLower(request.Header.Get("Accept")), "text/html") {
		writer.Header().Set("Content-Type", "text/html")
		writer.WriteHeader(status)

		css, body, errEnc := encoder.Encode(data)
		if errEnc != nil {
			responseErr(writer, request, http.StatusInternalServerError, errEnc, "encoder failed")

			return
		}

		if errExec := indexTMPL.Execute(writer, map[string]any{
			"title": title,
			"css":   template.CSS(css),
			"body":  template.HTML(body), //nolint:gosec
		}); errExec != nil {
			slog.Error("failed to execute template", ErrAttr(errExec))
		}

		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)

	if errWrite := encodeJSONIndent(writer, data); errWrite != nil {
		slog.Error("failed to write out json response", ErrAttr(errWrite))
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

func steamIDFromSlug(w http.ResponseWriter, r *http.Request) (steamid.SteamID, bool) {
	sid64 := steamid.New(r.PathValue("steam_id"))
	if !sid64.Valid() {
		responseErr(w, r, http.StatusNotFound, errInvalidSteamID, "")

		return steamid.SteamID{}, false
	}

	return sid64, true
}

func initTemplate() error {
	tmplProfiles, errTmpl := template.New("").Parse(`<!DOCTYPE html>
		<html>
		<head>
			<title>{{ .title }}</title>
			<style> body {background-color: #272822;} {{ .css }} </style>
		</head>
		<body>{{ .body }}</body>
		</html>`)

	if errTmpl != nil {
		return errors.Join(errTmpl, errParseTemplate)
	}

	indexTMPL = tmplProfiles

	return nil
}

func createRouter(database *pgStore, cacheHandler cache) (*http.ServeMux, error) {
	encoder = newStyleEncoder()
	if errTmpl := initTemplate(); errTmpl != nil {
		return nil, errTmpl
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /log/{log_id}", handleGetLogByID(database))
	mux.HandleFunc("GET /bans", handleGetBans())
	mux.HandleFunc("GET /summary", handleGetSummary(cacheHandler))
	mux.HandleFunc("GET /profile", handleGetProfile(database, cacheHandler))
	mux.HandleFunc("GET /comp", handleGetComp(cacheHandler))
	mux.HandleFunc("GET /friends", handleGetFriendList(cacheHandler))
	mux.HandleFunc("GET /sourcebans", handleGetSourceBansMany(database))
	mux.HandleFunc("GET /sourcebans/{steam_id}", handleGetSourceBans(database))
	mux.HandleFunc("GET /bd", handleGetBotDetector(database))
	mux.HandleFunc("GET /log/player/{steam_id}", handleGetLogsSummary(database))
	mux.HandleFunc("GET /log/player/{steam_id}/list", handleGetLogsList(database))
	mux.HandleFunc("GET /serveme", handleGetServemeList(database))
	mux.HandleFunc("GET /steamid/{steam_id}", handleGetSteamID())
	mux.HandleFunc("GET /", handleGetIndex())

	return mux, nil
}

func newHTTPServer(ctx context.Context, router *http.ServeMux, addr string) *http.Server {
	httpServer := &http.Server{ //nolint:exhaustruct
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  apiTimeout,
		WriteTimeout: apiTimeout,
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
	}

	return httpServer
}

func runHTTP(ctx context.Context, router *http.ServeMux, listenAddr string) int {
	httpServer := newHTTPServer(ctx, router, listenAddr)

	go func() {
		//goland:noinspection ALL
		if strings.HasPrefix(listenAddr, ":") {
			listenAddr = "localhost" + listenAddr
		}
		slog.Info("Starting HTTP service", slog.String("address", "http://"+listenAddr))
		if errServe := httpServer.ListenAndServe(); errServe != nil && !errors.Is(errServe, http.ErrServerClosed) {
			slog.Error("error trying to shutdown http service", ErrAttr(errServe))
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if errShutdown := httpServer.Shutdown(shutdownCtx); errShutdown != nil { //nolint:contextcheck
		slog.Error("Error shutting down http service", ErrAttr(errShutdown))

		return 1
	}

	return 0
}
