package main

import (
	"context"
	"errors"
	"log/slog"
)

var version = "1.0.7"

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

func runLogsTFScraper(ctx context.Context, database *pgStore, config appConfig) error {
	logsScraper, errLogsTF := NewLogsTFScraper(database, config)
	if errLogsTF != nil {
		return errLogsTF
	}

	if config.ProxiesEnabled {
		if errProxies := attachCollectorProxies(logsScraper.Collector, &config); errProxies != nil {
			return errProxies
		}
	}

	go startLogsTF(ctx, logsScraper)

	return nil
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

	go startScrapers(ctx, database, scrapers)

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

	if config.LogstfScraperEnabled {
		if err := runLogsTFScraper(ctx, database, config); err != nil {
			slog.Error("failed to init logstf scraper", ErrAttr(err))

			return 1
		}
	}

	if config.RGLScraperEnabled {
		rglScraper := NewRGLScraper(database)
		go rglScraper.start(ctx)
	}

	if config.ETF2LScraperEnabled {
		etf2lScraper := NewETF2LScraper(database)
		go etf2lScraper.start(ctx)
	}

	if config.SourcebansScraperEnabled {
		if err := runSourcebansScraper(ctx, database, config); err != nil {
			slog.Error("failed to init sourcebans scraper", ErrAttr(err))

			return 1
		}
	}

	go listUpdater(ctx, database)
	go profileUpdater(ctx, database)
	go startServeMeUpdater(ctx, database)

	return runHTTP(ctx, router, config.ListenAddr)
}

func main() {
	execute()
}
