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

// Profile is a high level meta profile of several services.
type Profile struct {
	Summary   steamweb.PlayerSummary  `json:"summary"`
	BanState  steamweb.PlayerBanState `json:"ban_state"`
	Seasons   []Season                `json:"seasons"`
	Friends   []steamweb.Friend       `json:"friends"`
	LogsCount int64                   `json:"logs_count"`
}

func (a *App) loadProfile(ctx context.Context, log *zap.Logger, steamID steamid.SID64, profile *Profile) error {
	sum, errSum := a.getSteamSummary(ctx, steamID)
	if errSum != nil || len(sum) == 0 {
		return errSum
	}

	profile.Summary = sum[0]

	banState, errBanState := steamweb.GetPlayerBans(ctx, steamid.Collection{steamID})
	if errBanState != nil || len(banState) == 0 {
		return errors.Wrap(errBanState, "Failed to query player bans")
	}

	profile.BanState = banState[0]

	_, errFriends := a.getSteamFriends(ctx, steamID)
	if errFriends != nil {
		log.Debug("Failed to get friends", zap.Error(errors.Wrap(errFriends, "Failed to get friends")))
	}
	// profile.Friends = friends

	sort.Slice(profile.Seasons, func(i, j int) bool {
		return profile.Seasons[i].DivisionInt < profile.Seasons[j].DivisionInt
	})

	return nil
}

type styleEncoder struct {
	style     *chroma.Style
	formatter *html.Formatter
	lexer     chroma.Lexer
}

func newStyleEncoder() *styleEncoder {
	newStyle := styles.Get("monokai")
	if newStyle == nil {
		newStyle = styles.Fallback
	}

	return &styleEncoder{
		style:     newStyle,
		formatter: html.New(html.WithClasses(true)),
		lexer:     lexers.Get("json"),
	}
}

func (s *styleEncoder) Encode(value any) (string, string, error) {
	jsonBody, errJSON := json.MarshalIndent(value, "", "    ")
	if errJSON != nil {
		return "", "", errors.Wrap(errJSON, "Failed to generate json")
	}

	iterator, errTokenize := s.lexer.Tokenise(nil, string(jsonBody))
	if errTokenize != nil {
		return "", "", errors.Wrap(errTokenize, "Failed to tokenise json")
	}

	cssBuf := bytes.NewBuffer(nil)
	if errWrite := s.formatter.WriteCSS(cssBuf, s.style); errWrite != nil {
		return "", "", errors.Wrap(errWrite, "Failed to generate HTML")
	}

	bodyBuf := bytes.NewBuffer(nil)
	if errFormat := s.formatter.Format(bodyBuf, s.style, iterator); errFormat != nil {
		return "", "", errors.Wrap(errFormat, "Failed to format json")
	}

	return cssBuf.String(), bodyBuf.String(), nil
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

func renderSyntax(ctx *gin.Context, encoder *styleEncoder, value any, tmpl string, args syntaxTemplate) {
	if !strings.Contains(strings.ToLower(ctx.GetHeader("Accept")), "text/html") {
		ctx.JSON(http.StatusOK, value)

		return
	}

	css, body, errEncode := encoder.Encode(value)
	if errEncode != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, "Failed to load profile")

		return
	}

	args.setCSS(css)
	args.setBody(body)
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

func (a *App) createRouter() (*gin.Engine, error) {
	tmplProfiles, errTmpl := template.New("profiles").Parse(`<!DOCTYPE html>
<html>
<head> 
	<title>{{ .Title }}</title>
	<style> body {background-color: #272822;} {{ .CSS }} </style>
</head>
<body>{{ .Body }}</body>
</html>`)
	if errTmpl != nil {
		return nil, errors.Wrap(errTmpl, "Failed to parse html template")
	}

	gin.SetMode(a.config.RunMode)

	engine := gin.New()
	engine.SetHTMLTemplate(tmplProfiles)
	engine.Use(apiErrorHandler(a.log), gin.Recovery())
	engine.GET("/bans", a.handleGetBans())
	engine.GET("/summary", a.handleGetSummary())
	engine.GET("/profile", a.handleGetProfile())
	engine.GET("/profiles/:steam_id", a.handleGetProfiles())
	engine.GET("/sourcebans/:steam_id", a.handleGetSourceBans())

	return engine, nil
}

func (a *App) startAPI(ctx context.Context, addr string) error {
	const apiHandlerTimeout = 10 * time.Second

	const shutdownTimeout = 10 * time.Second

	log := a.log.Named("api")

	defer log.Info("Service status changed", zap.String("state", "stopped"))

	httpServer := &http.Server{ //nolint:exhaustruct
		Addr:         addr,
		Handler:      a.router,
		ReadTimeout:  apiHandlerTimeout,
		WriteTimeout: apiHandlerTimeout,
	}

	log.Info("Service status changed", zap.String("state", "ready"))

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if errShutdown := httpServer.Shutdown(shutdownCtx); errShutdown != nil { //nolint:contextcheck
			log.Error("Error shutting down http service", zap.Error(errShutdown))
		}
	}()

	return errors.Wrap(httpServer.ListenAndServe(), "Error returned from HTTP server")
}
