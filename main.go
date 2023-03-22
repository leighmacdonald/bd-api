package main

import (
	"context"
	"github.com/leighmacdonald/steamweb"
	"go.uber.org/zap"
	"io"
	"net/http"
	"os"
	"regexp"
)

var logger *zap.Logger

var cacheDir string

func main() {
	ctx := context.Background()
	defer func() {
		logger.Info("Shutting down")
		if errSync := logger.Sync(); errSync != nil {
			logger.Panic("Failed to sync", zap.Error(errSync))
		}
	}()
	listenAddr, found := os.LookupEnv("LISTEN")
	if !found {
		listenAddr = ":8888"
	}
	steamAPIKey, steamAPIKeyFound := os.LookupEnv("STEAM_API_KEY")
	if !steamAPIKeyFound || steamAPIKey == "" {
		logger.Panic("Must set STEAM_API_KEY")
	}
	if errSetKey := steamweb.SetKey(steamAPIKey); errSetKey != nil {
		logger.Panic("Failed to configure steam api key", zap.Error(errSetKey))
	}

	cache := newCaches(ctx, steamCacheTimeout, compCacheTimeout, steamCacheTimeout)

	http.HandleFunc("/bans", limit(getHandler(handleGetBans(cache))))
	http.HandleFunc("/summary", limit(getHandler(handleGetSummary(cache))))
	http.HandleFunc("/profile", limit(getHandler(handleGetProfile(cache))))

	logger.Info("Starting HTTP listener", zap.String("host", listenAddr))
	if errServe := http.ListenAndServe(listenAddr, nil); errServe != nil {
		logger.Error("HTTP listener error", zap.Error(errServe))
	}
}

func logClose(closer io.Closer) {
	if err := closer.Close(); err != nil {
		logger.Error("Failed to close", zap.Error(err))
	}
}

func init() {
	cacheDir = "./cache"
	logConfig := zap.NewDevelopmentConfig()
	newLogger, errLogger := logConfig.Build()
	if errLogger != nil {
		panic(errLogger)
	}
	logger = newLogger
	steamAPIKey, steamAPIKeyFound := os.LookupEnv("STEAM_API_KEY")
	if !steamAPIKeyFound || steamAPIKey == "" {
		panic("Must set STEAM_API_KEY")
	}
	if errSetKey := steamweb.SetKey(steamAPIKey); errSetKey != nil {
		panic(errSetKey)
	}
	reLOGSResults = regexp.MustCompile(`<p>(\d+|\d+,\d+)\sresults</p>`)
	//reETF2L = regexp.MustCompile(`.org/forum/user/(\d+)`)
	reUGCRank = regexp.MustCompile(`Season (\d+) (\D+) (\S+)`)
}
