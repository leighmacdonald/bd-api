package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"hash/crc32"
	"html/template"
	"io"
	"log/slog"
	"net"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/leighmacdonald/steamid/v4/steamid"
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

func createRouter(database *pgStore, cacheHandler cache, config appConfig) (*http.ServeMux, error) {
	encoder = newStyleEncoder()
	if errTmpl := initTemplate(); errTmpl != nil {
		return nil, errTmpl
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /log/{log_id}", handleGetLogByID(database))
	mux.HandleFunc("GET /bans", handleGetBans())
	mux.HandleFunc("GET /summary", handleGetSummary(cacheHandler))
	mux.HandleFunc("GET /profile", handleGetProfile(database, cacheHandler))
	mux.HandleFunc("GET /friends", handleGetFriendList(cacheHandler))
	mux.HandleFunc("GET /owned_games", handleGetOwnedGames(database))
	mux.HandleFunc("GET /sourcebans", handleGetSourceBansMany(database))
	mux.HandleFunc("GET /sourcebans/{steam_id}", handleGetSourceBans(database))
	mux.HandleFunc("GET /bd", handleGetBotDetector(database))
	mux.HandleFunc("GET /log/player/{steam_id}", handleGetLogsSummary(database))
	mux.HandleFunc("GET /log/player/{steam_id}/list", handleGetLogsList(database))
	mux.HandleFunc("GET /serveme", handleGetServemeList(database))
	mux.HandleFunc("GET /steamid/{steam_id}", handleGetSteamID())
	mux.HandleFunc("GET /", handleGetIndex())
	mux.HandleFunc("GET /stats", handleGetStats(database))
	mux.HandleFunc("GET /list/rgl", handleGetRGLList(database, config))
	mux.HandleFunc("GET /list/etf2l", handleGetETF2LList(database, config))
	mux.HandleFunc("GET /list/serveme", handleGetServemeListBD(database, config))
	mux.HandleFunc("GET /rgl/player_history", handleGetRGLPlayerHistory(database))
	mux.HandleFunc("GET /league_bans", handleGetLeagueBans(database))

	return mux, nil
}

type apiErr struct {
	Error string `json:"error"`
}

func responseErr(w http.ResponseWriter, request *http.Request, status int, err error, userMsg string) {
	msg := err.Error()
	if userMsg != "" {
		msg = userMsg
	}
	renderResponse(w, request, status, map[string]string{"error": msg}, "Error")
	slog.Error("Error executing request", ErrAttr(err),
		slog.String("path", request.URL.Path), slog.String("query", request.URL.RawQuery))
}

func responseOk(w http.ResponseWriter, r *http.Request, data any, title string) {
	renderResponse(w, r, http.StatusOK, data, title)
}

func renderResponse(writer http.ResponseWriter, request *http.Request, status int, data any, title string) {
	if data == nil {
		data = []string{}
	}

	// steamids and logs never change, use very long cache timeout
	perm := strings.HasPrefix(request.URL.Path, "/steamid/") ||
		(strings.HasPrefix(request.URL.Path, "/log/") && !strings.HasPrefix(request.URL.Path, "/log/player"))

	if strings.Contains(strings.ToLower(request.Header.Get("Accept")), "text/html") {
		renderHTMLResponse(writer, request, status, data, perm, title)
	} else {
		renderJSONResponse(writer, request, status, data, perm)
	}
}

func renderHTMLResponse(writer http.ResponseWriter, request *http.Request, status int, data any, perm bool, title string) {
	writer.Header().Set("Content-Type", "text/html")

	css, body, errEnc := encoder.Encode(data)
	if errEnc != nil {
		responseErr(writer, request, http.StatusInternalServerError, errEnc, "encoder failed")

		return
	}

	buf := bytes.NewBuffer(nil)

	if errExec := indexTMPL.Execute(buf, map[string]any{
		"title": title,
		"css":   template.CSS(css),
		"body":  template.HTML(body), //nolint:gosec
	}); errExec != nil {
		slog.Error("Failed to execute template", ErrAttr(errExec))
	}

	if handleCacheHeaders(writer, request, buf.Bytes(), perm) {
		return
	}

	writer.WriteHeader(status)

	if _, err := io.Copy(writer, buf); err != nil {
		slog.Error("Failed to copy response buffer", ErrAttr(err))
	}
}

func renderJSONResponse(writer http.ResponseWriter, request *http.Request, status int, data any, perm bool) {
	buf := bytes.NewBuffer(nil)

	if errWrite := encodeJSONIndent(buf, data); errWrite != nil {
		slog.Error("Failed to write out json response", ErrAttr(errWrite))
	}

	writer.Header().Set("Content-Type", "application/json")

	if handleCacheHeaders(writer, request, buf.Bytes(), perm) {
		return
	}

	writer.WriteHeader(status)

	if _, err := io.Copy(writer, buf); err != nil {
		slog.Error("Failed to copy response buffer", ErrAttr(err))
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

// getSteamIDs parses the steamids url query value for valid steamids and returns them as a steamid.Collection.
//
// If there is only one entry provided, then it will also attempt resolve the vanity name and/or parse the profile url.
// This is to prevent over use of the API since many api endpoints are able to accept up to 100 steamids at a time while
// the vanity resolving endpoint only supports a single entry at a time.
func getSteamIDs(writer http.ResponseWriter, request *http.Request) (steamid.Collection, bool) {
	steamIDQuery := request.URL.Query().Get("steamids")

	if steamIDQuery == "" {
		responseErr(writer, request, http.StatusBadRequest, errInvalidQueryParams, "")

		return nil, false
	}

	entries := strings.Split(steamIDQuery, ",")

	if len(entries) == 1 {
		sid64, errResolve := steamid.Resolve(request.Context(), entries[0])
		if errResolve != nil {
			responseErr(writer, request, http.StatusBadRequest, errInvalidSteamID, "")

			return nil, false
		}

		return steamid.Collection{sid64}, true
	}

	// Sort sids so that etags are more accurate
	slices.Sort(entries)

	var validIDs steamid.Collection

	for _, steamID := range entries {
		sid64 := steamid.New(steamID)
		if !sid64.Valid() {
			responseErr(writer, request, http.StatusBadRequest, errInvalidSteamID, "Invalid steamid/profile")

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
		responseErr(writer, request, http.StatusBadRequest, errTooMany, "Max 100 steamids allowed")

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

func boolQuery(request *http.Request, name string) (bool, bool) {
	boolString := request.URL.Query().Get(name)
	if boolString == "" {
		return false, true
	}

	boolVal, err := strconv.ParseBool(boolString)
	if err != nil {
		return false, false
	}

	return boolVal, true
}

func steamIDFromSlug(w http.ResponseWriter, r *http.Request) (steamid.SteamID, bool) {
	sid64, errResolve := steamid.Resolve(r.Context(), r.PathValue("steam_id"))
	if errResolve != nil {
		responseErr(w, r, http.StatusNotFound, errInvalidSteamID, "Could not resolve steamid")

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

func generateETag(data []byte) string {
	return fmt.Sprintf(`%08X`, crc32.ChecksumIEEE(data))
}

func handleCacheHeaders(writer http.ResponseWriter, request *http.Request, data []byte, perm bool) bool {
	eTag := generateETag(data)

	writer.Header().Set("ETag", eTag)

	if perm {
		writer.Header().Set("Cache-Control", "max-age=31536000") // One year cache
	} else {
		writer.Header().Set("Cache-Control", "max-age=86400") // One day cache
	}

	if match := request.Header.Get("If-None-Match"); match != "" {
		if strings.Contains(match, eTag) {
			writer.WriteHeader(http.StatusNotModified)

			return true
		}
	}

	return false
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
