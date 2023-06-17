package main

import (
	"context"
	"go.uber.org/zap"
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
	if errConfig := readConfig(&config); errConfig != nil {
		logger.Panic("Failed to load config", zap.Error(errConfig))
	}
	if !exists(cacheDir) {
		if errMkDir := os.MkdirAll(cacheDir, 0755); errMkDir != nil {
			logger.Fatal("Failed to create cache dir", zap.String("dir", cacheDir), zap.Error(errMkDir))
		}
	}
	db, errDB := newStore(ctx, config.DSN)
	if errDB != nil {
		logger.Fatal("Failed to connect to database", zap.Error(errDB))
	}
	if config.SourcebansScraperEnabled {
		scrapers := createScrapers()
		if errInitScrapers := initScrapers(ctx, db, scrapers); errInitScrapers != nil {
			logger.Fatal("Failed to initialize scrapers", zap.Error(errInitScrapers))
		}
		go startScrapers(ctx, &config, scrapers, db)
	}
	if errAPI := startAPI(ctx, createRouter(db), config.ListenAddr); errAPI != nil {
		logger.Error("HTTP server returned error", zap.Error(errAPI))
	}
}

var (
	logger   *zap.Logger
	cacheDir = "./.cache/"
)

func init() {
	logCfg := zap.NewProductionConfig()
	logCfg.DisableStacktrace = true
	newLogger, errLogger := logCfg.Build()
	if errLogger != nil {
		panic(errLogger)
	}
	logger = newLogger
}
