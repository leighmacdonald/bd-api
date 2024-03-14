package main

import (
	"context"
	"log/slog"
)

func createAppDeps(ctx context.Context) (appConfig, cache, *pgStore, error) {
	var config appConfig

	if errConfig := readConfig(&config); errConfig != nil {
		return config, nil, nil, errConfig
	}

	loggerClose := MustCreateLogger(config.LogFilePath, config.LogLevel, config.LogFileEnabled)
	defer loggerClose()

	cacheHandler, errCache := createCache(config.EnableCache, config.CacheDir)
	if errCache != nil {
		return config, nil, nil, errCache
	}

	database, errDB := newStore(ctx, config.DSN)
	if errDB != nil {
		return config, nil, nil, errDB
	}
	if errPing := database.pool.Ping(ctx); errPing != nil {
		return config, nil, nil, errPing
	}

	return config, cacheHandler, database, nil
}

func run(ctx context.Context) int {
	config, cacheHandler, database, errSetup := createAppDeps(ctx)
	if errSetup != nil {
		slog.Error("failed to setup app dependencies", ErrAttr(errSetup))
		return 1
	}

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

	go listUpdater(ctx, database)
	go profileUpdater(ctx, database)

	return runHTTP(ctx, router, config.ListenAddr)
}

func main() {
	execute()
}
