package main

import (
	"context"
	"errors"
	"log/slog"
)

var version = "1.1.0"

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

	if err := setupQueue(ctx, database.pool); err != nil {
		return config, nil, nil, errors.Join(err, errDatabaseMigrate)
	}

	return config, cacheHandler, database, nil
}

func runSourcebansScraper(ctx context.Context, database *pgStore, config appConfig) error {
	scrapers, errScrapers := initScrapers(ctx, database, config.CacheDir)
	if errScrapers != nil {
		return errScrapers
	}

	if config.ProxiesEnabled {
		for _, scraper := range scrapers {
			if errProxies := attachCollectorProxies(scraper.Collector, &config); errProxies != nil {
				return errProxies
			}
		}
	}

	runScrapers(ctx, database, scrapers)

	return nil
}

func run(ctx context.Context) int {
	config, cacheHandler, database, errSetup := createAppDeps(ctx)
	if errSetup != nil {
		slog.Error("failed to setup app dependencies", ErrAttr(errSetup))

		return 1
	}

	router, errRouter := createRouter(database, cacheHandler, config)
	if errRouter != nil {
		slog.Error("failed to create router", ErrAttr(errRouter))

		return 1
	}

	proxyMgr := NewProxyManager()
	if config.ProxiesEnabled {
		proxyMgr.start(&config)

		defer proxyMgr.stop()
	}

	jobClient, err := initJobClient(ctx, database, config)
	if err != nil {
		slog.Error("Failed to create job client", ErrAttr(err))

		return 1
	}

	defer func() {
		if errStop := jobClient.Stop(ctx); errStop != nil {
			slog.Error("Failed to cleanly stop job client", ErrAttr(errStop))
		}
	}()

	return runHTTP(ctx, router, config.ListenAddr)
}

func main() {
	execute()
}
