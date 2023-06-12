package main

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/leighmacdonald/steamweb"
	"go.uber.org/zap"
	"html/template"
	"net/http"
	"sort"
	"time"
)

func handleGetSummary() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		steamIDQuery, ok := ctx.GetQuery("steam_id")
		if !ok {
			ctx.AbortWithStatus(http.StatusBadRequest)
			return
		}
		sid64, steamIDErr := steamid.SID64FromString(steamIDQuery)
		if steamIDErr != nil || !sid64.Valid() {
			ctx.AbortWithStatus(http.StatusBadRequest)
			return
		}
		ids := steamid.Collection{sid64}
		summaries, errSum := getSteamSummary(sid64)
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
		bans, errBans := steamweb.GetPlayerBans(ids)
		if errBans != nil || len(ids) != len(bans) {
			logger.Error("Failed to fetch player bans",
				zap.Error(errBans), zap.Int64("steam_id", sid.Int64()))
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, "Failed to fetch bans")
			return
		}
		ctx.JSON(http.StatusOK, bans)
	}
}

// Profile is a high level meta profile of several services
type Profile struct {
	Summary   steamweb.PlayerSummary  `json:"summary"`
	BanState  steamweb.PlayerBanState `json:"ban_state"`
	Seasons   []Season                `json:"seasons"`
	Friends   []steamweb.Friend       `json:"friends"`
	LogsCount int64                   `json:"logs_count"`
}

func loadProfile(steamID steamid.SID64, profile *Profile) error {
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
		if errProfile := loadProfile(sid64, &profile); errProfile != nil {
			logger.Error("Failed to load profile", zap.Error(errProfile))
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, "Failed to fetch profile")
			return
		}
		ctx.JSON(http.StatusOK, profile)
	}
}

func handleGetProfiles() gin.HandlerFunc {
	style := styles.Get("monokai")
	if style == nil {
		style = styles.Fallback
	}
	formatter := html.New(html.WithClasses(true))
	lexer := lexers.Get("json")
	type tmplArgs struct {
		name string
		css  template.HTML
		body template.HTML
	}

	return func(ctx *gin.Context) {
		slug := ctx.Param("steam_id")
		lCtx, cancel := context.WithTimeout(ctx, time.Second*10)
		defer cancel()
		sid64, errSid := steamid.ResolveSID64(lCtx, slug)
		if errSid != nil {
			ctx.AbortWithStatusJSON(http.StatusNotFound, "not found")
			return
		}
		var profile Profile
		if errProfile := loadProfile(sid64, &profile); errProfile != nil {
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, "Failed to load profile")
			return
		}
		jsonBody, errJSON := json.MarshalIndent(profile, "", "    ")
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
		ctx.HTML(http.StatusOK, "profiles", tmplArgs{
			name: profile.Summary.PersonaName,
			css:  template.HTML(cssBuf.String()),
			body: template.HTML(bodyBuf.String()),
		})
	}
}

func getHandler(wrappedFn func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		wrappedFn(w, req)
	}
}

func ErrorHandler(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		for _, ginErr := range c.Errors {
			logger.Error("Unhandled HTTP Error", zap.Error(ginErr))
		}
	}
}

func createRouter() *gin.Engine {
	tmplProfiles, errTmpl := template.New("profiles").Parse(`
<!DOCTYPE html>
<html>
<head>
	<title>{{ .name }}</title>
	<style> body {background-color: #272822;} {{ .style }} </style>
</head>
<body>{{ .body }}</body>
</html>`)
	if errTmpl != nil {
		logger.Panic("Failed to parse html template", zap.Error(errTmpl))
	}

	engine := gin.New()
	engine.SetHTMLTemplate(tmplProfiles)
	engine.Use(ErrorHandler(logger), gin.Recovery())
	engine.GET("/bans", handleGetBans())
	engine.GET("/summary", handleGetSummary())
	engine.GET("/profile", handleGetProfile())
	engine.GET("/profiles/:steam_id", handleGetProfiles())
	return engine
}

func startAPI(ctx context.Context, addr string) error {
	httpServer := &http.Server{
		Addr:           addr,
		Handler:        createRouter(),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	logger.Info("Service status changed", zap.String("state", "ready"))
	defer logger.Info("Service status changed", zap.String("state", "stopped"))
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		if errShutdown := httpServer.Shutdown(shutdownCtx); errShutdown != nil {
			logger.Error("Error shutting down http service", zap.Error(errShutdown))
		}
	}()
	return httpServer.ListenAndServe()
}
