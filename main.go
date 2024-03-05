package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var config appConfig

	if errConfig := readConfig(&config); errConfig != nil {
		panic(fmt.Sprintf("Failed to load config: %v", errConfig))
	}

	loggerClose := MustCreateLogger(config.LogFilePath, config.LogLevel, config.LogFileEnabled)
	defer loggerClose()

	slog.Info("Starting...")

	var cacheHandler cache

	if config.EnableCache {
		localCache, cacheErr := newFSCache(config.CacheDir)
		if cacheErr != nil {
			slog.Error("Failed to create fsCache", ErrAttr(cacheErr))

			return 1
		}

		cacheHandler = localCache
	} else {
		cacheHandler = &nopCache{}
	}

	database, errDB := newStore(ctx, config.DSN)
	if errDB != nil {
		slog.Error("Failed to instantiate database", ErrAttr(errDB))

		return 1
	}

	if errPing := database.pool.Ping(ctx); errPing != nil {
		slog.Error("failed to connect to database")

		return 1
	}

	pm := newProxyManager()

	router, errRouter := createRouter(config.RunMode, database, cacheHandler)
	if errRouter != nil {
		slog.Error("failed to create router", ErrAttr(errRouter))

		return 1
	}

	if config.SourcebansScraperEnabled {
		scrapers, errScrapers := initScrapers(ctx, database, config.CacheDir)
		if errScrapers != nil {
			slog.Error("failed to setup scrapers")

			return 1
		}

		go startScrapers(ctx, config, pm, database, scrapers)
	}

	go profileUpdater(ctx, database)

	return runHTTP(ctx, router, config.ListenAddr)
}

func main() {
	os.Exit(run())
}
