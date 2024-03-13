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
	gCtx := context.Background()
	ctx, stop := signal.NotifyContext(gCtx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	var config appConfig

	if errConfig := readConfig(&config); errConfig != nil {
		panic(fmt.Sprintf("Failed to load config: %v", errConfig))
	}

	loggerClose := MustCreateLogger(config.LogFilePath, config.LogLevel, config.LogFileEnabled)
	defer loggerClose()

	cacheHandler, errCache := createCache(config.EnableCache, config.CacheDir)
	if errCache != nil {
		slog.Error("failed to setup cache", ErrAttr(errCache))

		return 1
	}

	database, errDB := newStore(ctx, config.DSN)
	if errDB != nil {
		slog.Error("Failed to instantiate database", ErrAttr(errDB))

		return 2
	}

	if errPing := database.pool.Ping(ctx); errPing != nil {
		slog.Error("failed to connect to database")

		return 3
	}

	router, errRouter := createRouter(config.RunMode, database, cacheHandler)
	if errRouter != nil {
		slog.Error("failed to create router", ErrAttr(errRouter))

		return 4
	}

	if config.SourcebansScraperEnabled {
		scrapers, errScrapers := initScrapers(ctx, database, config.CacheDir)
		if errScrapers != nil {
			slog.Error("failed to setup scrapers")

			return 5
		}

		pm := newProxyManager()
		if config.ProxiesEnabled {
			pm.start(&config)

			defer pm.stop()

			for _, scraper := range scrapers {
				if errProxies := pm.setup(scraper.Collector, &config); errProxies != nil {
					slog.Error("Failed to setup proxies", ErrAttr(errProxies))

					continue
				}
			}
		}

		go startScrapers(ctx, database, scrapers)
	}

	go listUpdater(gCtx, database)
	go profileUpdater(ctx, database)

	return runHTTP(ctx, router, config.ListenAddr)
}

func main() {
	os.Exit(run())
}
