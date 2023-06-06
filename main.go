package main

import (
	"context"
	"github.com/leighmacdonald/steamweb"
	"go.uber.org/zap"
	"log"
	"net/http"
	"os"
	"regexp"
)

func main() {
	ctx = context.Background()
	defer func() {
		logger.Info("Shutting down")
		if errSync := logger.Sync(); errSync != nil {
			logger.Panic("Failed to sync", zap.Error(errSync))
		}
	}()

	var config Config
	if errConfig := readConfig("config.yml", &config); errConfig != nil {
		logger.Panic("Failed to load config", zap.Error(errConfig))
	}
	if errSetKey := steamweb.SetKey(config.SteamApiKey); errSetKey != nil {
		log.Panicf("Failed to set steam api key: %v\n", errSetKey)
	}
	if config.SteamApiKey == "" {
		logger.Panic("Must set STEAM_API_KEY")
	}
	if errSetKey := steamweb.SetKey(config.SteamApiKey); errSetKey != nil {
		logger.Panic("Failed to configure steam api key", zap.Error(errSetKey))
	}

	cache = newCaches(ctx, steamCacheTimeout, compCacheTimeout, steamCacheTimeout)

	if errServe := http.ListenAndServe(config.ListenAddr, nil); errServe != nil {
		log.Printf("HTTP Server returned error: %v", errServe)
	}
}

var (
	cache    caches
	ctx      context.Context
	logger   *zap.Logger
	cacheDir = "./.cache/"
)

func init() {
	newLogger, errLogger := zap.NewProduction()
	if errLogger != nil {
		panic(errLogger)
	}
	logger = newLogger

	if !exists(cacheDir) {
		if errMkDir := os.MkdirAll(cacheDir, 0755); errMkDir != nil {
			log.Fatal("Failed to create cache dir", zap.String("dir", cacheDir), zap.Error(errMkDir))
		}
	}
	cache = newCaches(ctx, steamCacheTimeout, compCacheTimeout, steamCacheTimeout)

	reLOGSResults = regexp.MustCompile(`<p>(\d+|\d+,\d+)\sresults</p>`)
	//reETF2L = regexp.MustCompile(`.org/forum/user/(\d+)`)
	reUGCRank = regexp.MustCompile(`Season (\d+) (\D+) (\S+)`)

}
