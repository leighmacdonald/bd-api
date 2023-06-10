package main

import (
	"context"
	"github.com/leighmacdonald/steamweb"
	"go.uber.org/zap"
	"log"
	"net/http"
	"os"
)

func main() {
	ctx := context.Background()
	defer func() {
		logger.Info("Shutting down")
		if errSync := logger.Sync(); errSync != nil {
			logger.Panic("Failed to sync", zap.Error(errSync))
		}
	}()
	logger.Info("Starting...")
	var config appConfig
	if errConfig := readConfig("config.yml", &config); errConfig != nil {
		logger.Panic("Failed to load config", zap.Error(errConfig))
	}
	if errSetKey := steamweb.SetKey(config.SteamAPIKey); errSetKey != nil {
		log.Panicf("Failed to set steam api key: %v\n", errSetKey)
	}
	if config.SteamAPIKey == "" {
		logger.Panic("Must set STEAM_API_KEY")
	}
	if errSetKey := steamweb.SetKey(config.SteamAPIKey); errSetKey != nil {
		logger.Panic("Failed to configure steam api key", zap.Error(errSetKey))
	}

	if !exists(cacheDir) {
		if errMkDir := os.MkdirAll(cacheDir, 0755); errMkDir != nil {
			log.Fatal("Failed to create cache dir", zap.String("dir", cacheDir), zap.Error(errMkDir))
		}
	}

	db, errDB := newStore(ctx, config.DSN)
	if errDB != nil {
		logger.Fatal("Failed to connect to database", zap.Error(errDB))
	}

	cache = newCaches(ctx, steamCacheTimeout, compCacheTimeout, steamCacheTimeout)
	if config.SourcebansScraperEnabled {
		scrapers := createScrapers()
		if errInitScrapers := initScrapers(ctx, db, scrapers); errInitScrapers != nil {
			logger.Fatal("Failed to initialize scrapers", zap.Error(errInitScrapers))
		}
		go startScrapers(&config, scrapers)

	}
	http.HandleFunc("/bans", getHandler(handleGetBans()))
	http.HandleFunc("/summary", getHandler(handleGetSummary()))
	http.HandleFunc("/profile", getHandler(handleGetProfile()))
	http.HandleFunc("/kick", onPostKick)
	http.HandleFunc(profilesSlugURL, getHandler(handleGetProfiles(ctx)))

	if errServe := http.ListenAndServe(config.ListenAddr, nil); errServe != nil {
		log.Printf("HTTP Server returned error: %v", errServe)
	}
}

var (
	cache    caches
	logger   *zap.Logger
	cacheDir = "./.cache/"
)

func init() {
	newLogger, errLogger := zap.NewProduction()
	if errLogger != nil {
		panic(errLogger)
	}
	logger = newLogger
}
