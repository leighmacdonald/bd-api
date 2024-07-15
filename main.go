package main

import (
	"context"
	"errors"
	"log/slog"
)

var version = "1.0.5"

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
	if err := database.pool.Ping(ctx); err != nil {
		return config, nil, nil, errors.Join(err, errDatabasePing)
	}

	return config, cacheHandler, database, nil
}

func run(ctx context.Context) int {
	config, cacheHandler, database, errSetup := createAppDeps(ctx)
	if errSetup != nil {
		slog.Error("failed to setup app dependencies", ErrAttr(errSetup))

		return 1
	}

	router, errRouter := createRouter(database, cacheHandler)
	if errRouter != nil {
		slog.Error("failed to create router", ErrAttr(errRouter))

		return 1
	}

	proxyMgr := newProxyManager()
	if config.ProxiesEnabled {
		proxyMgr.start(&config)

		defer proxyMgr.stop()
	}

	if config.LogstfScraperEnabled {
		logsScraper, errLogsTF := newLogsTFScraper(database, config)
		if errLogsTF != nil {
			slog.Error("failed to init logstf scraper", ErrAttr(errLogsTF))

			return 1
		}

		if config.ProxiesEnabled {
			if errProxies := attachCollectorProxies(logsScraper.Collector, &config); errProxies != nil {
				slog.Error("Failed to setup proxies", ErrAttr(errProxies))

				return 1
			}
		}

		go logsScraper.start(ctx)
	}

	if config.SourcebansScraperEnabled {
		scrapers, errScrapers := initScrapers(ctx, database, config.CacheDir)
		if errScrapers != nil {
			slog.Error("failed to setup scrapers")

			return 1
		}

		if config.ProxiesEnabled {
			for _, scraper := range scrapers {
				if errProxies := attachCollectorProxies(scraper.Collector, &config); errProxies != nil {
					slog.Error("Failed to setup proxies", ErrAttr(errProxies))

					return 1
				}
			}
		}

		go startScrapers(ctx, database, scrapers)
	}

	go listUpdater(ctx, database)
	go profileUpdater(ctx, database)
	go startServeMeUpdater(ctx, database)

	return runHTTP(ctx, router, config.ListenAddr)
}

func main() {
	execute()
}
