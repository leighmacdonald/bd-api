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
		slog.Error("Failed to connect to database", ErrAttr(errDB))
	}

	pm := newProxyManager()

	app, errApp := NewApp(config, database, cacheHandler, pm)
	if errApp != nil {
		slog.Error("failed to create app", ErrAttr(errApp))

		return 1
	}

	if errStart := app.Start(ctx); errStart != nil {
		slog.Error("App returned error", ErrAttr(errStart))

		return 1
	}

	return 0
}

func main() {
	os.Exit(run())
}
