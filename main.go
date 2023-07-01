package main

import (
	"context"

	"go.uber.org/zap"
)

func main() {
	ctx := context.Background()

	logCfg := zap.NewProductionConfig()
	logCfg.DisableStacktrace = true
	logger, errLogger := logCfg.Build()

	if errLogger != nil {
		panic(errLogger)
	}

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

	cache, cacheErr := newFSCache(logger, "./.cache/")
	if cacheErr != nil {
		logger.Panic("Failed to create fsCache", zap.Error(cacheErr))
	}

	database, errDB := newStore(ctx, logger, config.DSN)
	if errDB != nil {
		logger.Fatal("Failed to connect to database", zap.Error(errDB))
	}

	pm := newProxyManager(logger)

	app := NewApp(logger, config, database, cache, pm)

	if errStart := app.Start(ctx); errStart != nil {
		logger.Error("App returned error", zap.Error(errStart))
		return
	}
}
