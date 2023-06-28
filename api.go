package main

import (
	"bytes"
	"context"
	"encoding/json"
	"html/template"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func handleGetSummary() gin.HandlerFunc {
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
		summaries, errSum := getSteamSummary(ctx, sid64)

		if errSum != nil || len(ids) != len(summaries) {
			logger.Error("Failed to fetch summary",
				zap.Error(errSum), zap.Int64("steam_id", sid64.Int64()))
			ctx.AbortWithStatus(http.StatusBadRequest)

			return
		}

		ctx.JSON(http.StatusOK, summaries)
	}
}

func handleGetBans() gin.HandlerFunc {
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
			logger.Error("Failed to fetch player bans",
				zap.Error(errBans), zap.Int64("steam_id", sid.Int64()))
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, "Failed to fetch bans")

			return
		}

		ctx.JSON(http.StatusOK, bans)
	}
}

// Profile is a high level meta profile of several services.
type Profile struct {
	Summary   steamweb.PlayerSummary  `json:"summary"`
	BanState  steamweb.PlayerBanState `json:"ban_state"`
	Seasons   []Season                `json:"seasons"`
	Friends   []steamweb.Friend       `json:"friends"`
	LogsCount int64                   `json:"logs_count"`
}

func loadProfile(ctx context.Context, steamID steamid.SID64, profile *Profile) error {
	sum, errSum := getSteamSummary(ctx, steamID)
	if errSum != nil || len(sum) == 0 {
		return errSum
	}

	profile.Summary = sum[0]

	banState, errBanState := steamweb.GetPlayerBans(ctx, steamid.Collection{steamID})
	if errBanState != nil || len(banState) == 0 {
		return errors.Wrap(errBanState, "Failed to query player bans")
	}

	profile.BanState = banState[0]

	_, errFriends := getSteamFriends(ctx, steamID)
	if errFriends != nil {
		logger.Debug("Failed to get friends", zap.Error(errors.Wrap(errFriends, "Failed to get friends")))
	}
	// profile.Friends = friends

	sort.Slice(profile.Seasons, func(i, j int) bool {
		return profile.Seasons[i].DivisionInt < profile.Seasons[j].DivisionInt
	})

	return nil
}

func handleGetProfile() gin.HandlerFunc {
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
		if errProfile := loadProfile(ctx, sid64, &profile); errProfile != nil {
			logger.Error("Failed to load profile", zap.Error(errProfile))
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, "Failed to fetch profile")

			return
		}

		ctx.JSON(http.StatusOK, profile)
	}
}

var (
	style     *chroma.Style
	formatter *html.Formatter
	lexer     chroma.Lexer
)

func init() {
	newStyle := styles.Get("monokai")
	if newStyle == nil {
		newStyle = styles.Fallback
	}

	style = newStyle

	formatter = html.New(html.WithClasses(true))
	lexer = lexers.Get("json")
}

type syntaxTemplate interface {
	setCSS(css string)
	setBody(css string)
}

type baseTmplArgs struct {
	CSS   template.CSS
	Body  template.HTML
	Title string
}

func (t *baseTmplArgs) setCSS(css string) {
	t.CSS = template.CSS(css)
}

func (t *baseTmplArgs) setBody(html string) {
	t.Body = template.HTML(html) //nolint:gosec
}

func handleGetSourceBans(database *pgStore) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		sid, errSid := steamIDFromSlug(ctx)
		if errSid != nil {
			return
		}

		bans, errBans := database.sbGetBansBySID(ctx, sid)
		if errBans != nil {
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, "Failed to query records")

			return
		}

		renderSyntax(ctx, bans, "profiles", &baseTmplArgs{ //nolint:exhaustruct
			Title: "source bans",
		})
	}
}

func steamIDFromSlug(ctx *gin.Context) (steamid.SID64, error) {
	const resolveTimeout = time.Second * 10

	slug := ctx.Param("steam_id")
	lCtx, cancel := context.WithTimeout(ctx, resolveTimeout)

	defer cancel()

	sid64, errSid := steamid.ResolveSID64(lCtx, slug)
	if errSid != nil {
		ctx.AbortWithStatusJSON(http.StatusNotFound, "not found")

		return "", errors.Wrap(errSid, "Failed to resolve steam id")
	}

	return sid64, nil
}

func handleGetProfiles() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		sid, errSid := steamIDFromSlug(ctx)
		if errSid != nil {
			return
		}

		var profile Profile
		if errProfile := loadProfile(ctx, sid, &profile); errProfile != nil {
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, "Failed to load profile")

			return
		}

		renderSyntax(ctx, profile, "profiles", &baseTmplArgs{ //nolint:exhaustruct
			Title: profile.Summary.PersonaName,
		})
	}
}

func renderSyntax(ctx *gin.Context, value any, tmpl string, args syntaxTemplate) {
	if !strings.Contains(strings.ToLower(ctx.GetHeader("Accept")), "text/html") {
		ctx.JSON(http.StatusOK, value)

		return
	}

	jsonBody, errJSON := json.MarshalIndent(value, "", "    ")
	if errJSON != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, "Failed to generate json")

		return
	}

	iterator, errTokenize := lexer.Tokenise(nil, string(jsonBody))
	if errTokenize != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, "Failed to tokenise json")

		return
	}

	cssBuf := bytes.NewBuffer(nil)
	if errWrite := formatter.WriteCSS(cssBuf, style); errWrite != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, "Failed to generate HTML")
	}

	bodyBuf := bytes.NewBuffer(nil)
	if errFormat := formatter.Format(bodyBuf, style, iterator); errFormat != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, "Failed to format json")

		return
	}

	args.setCSS(cssBuf.String())
	args.setBody(bodyBuf.String())
	ctx.HTML(http.StatusOK, tmpl, args)
}

func apiErrorHandler(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		for _, ginErr := range c.Errors {
			logger.Error("Unhandled HTTP Error", zap.Error(ginErr))
		}
	}
}

func createRouter(database *pgStore) *gin.Engine {
	tmplProfiles, errTmpl := template.New("profiles").Parse(`<!DOCTYPE html>
<html>
<head> 
	<title>{{ .Title }}</title>
	<style> body {background-color: #272822;} {{ .CSS }} </style>
</head>
<body>{{ .Body }}</body>
</html>`)
	if errTmpl != nil {
		logger.Panic("Failed to parse html template", zap.Error(errTmpl))
	}

	engine := gin.New()
	engine.SetHTMLTemplate(tmplProfiles)
	engine.Use(apiErrorHandler(logger), gin.Recovery())
	engine.GET("/bans", handleGetBans())
	engine.GET("/summary", handleGetSummary())
	engine.GET("/profile", handleGetProfile())
	engine.GET("/profiles/:steam_id", handleGetProfiles())
	engine.GET("/sourcebans/:steam_id", handleGetSourceBans(database))

	return engine
}

func startAPI(ctx context.Context, router *gin.Engine, addr string) error {
	const apiHandlerTimeout = 10 * time.Second

	const shutdownTimeout = 10 * time.Second

	defer logger.Info("Service status changed", zap.String("state", "stopped"))

	httpServer := &http.Server{ //nolint:exhaustruct
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  apiHandlerTimeout,
		WriteTimeout: apiHandlerTimeout,
	}

	logger.Info("Service status changed", zap.String("state", "ready"))

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if errShutdown := httpServer.Shutdown(shutdownCtx); errShutdown != nil { //nolint:contextcheck
			logger.Error("Error shutting down http service", zap.Error(errShutdown))
		}
	}()

	return errors.Wrap(httpServer.ListenAndServe(), "Error returned from HTTP server")
}
