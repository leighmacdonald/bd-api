package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func MustCreateLogger(conf appConfig) *zap.Logger {
	var loggingConfig zap.Config

	switch conf.RunMode {
	case gin.ReleaseMode:
		loggingConfig = zap.NewProductionConfig()
		loggingConfig.DisableCaller = true
	case gin.DebugMode:
		loggingConfig = zap.NewDevelopmentConfig()
		loggingConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	case gin.TestMode:
		return zap.NewNop()
	default:
		panic(fmt.Sprintf("Unknown run mode: %s", conf.RunMode))
	}

	if conf.LogFileEnabled {
		if exists(conf.LogFilePath) {
			if err := os.Remove(conf.LogFilePath); err != nil {
				panic(fmt.Sprintf("Failed to remove log file: %v", err))
			}
		}

		loggingConfig.OutputPaths = append(loggingConfig.OutputPaths, conf.LogFilePath)
	}

	level, errLevel := zap.ParseAtomicLevel(conf.LogLevel)
	if errLevel != nil {
		panic(fmt.Sprintf("Failed to parse log level: %v", errLevel))
	}

	loggingConfig.Level.SetLevel(level.Level())

	l, errLogger := loggingConfig.Build()
	if errLogger != nil {
		panic("Failed to create log config")
	}

	return l.Named("gb")
}

func main() {
	startTime := time.Now()
	ctx := context.Background()

	var config appConfig

	if errConfig := readConfig(&config); errConfig != nil {
		panic(fmt.Sprintf("Failed to load config: %v", errConfig))
	}

	logger := MustCreateLogger(config)

	defer func() {
		defer logger.Info("Exited", zap.Duration("uptime", time.Since(startTime)))

		if !config.LogFileEnabled {
			return
		}

		if errSync := logger.Sync(); errSync != nil {
			logger.Panic("Failed to sync", zap.Error(errSync))
		}
	}()

	logger.Info("Starting...")

	var cacheHandler cache

	if config.EnableCache {
		localCache, cacheErr := newFSCache(logger, config.CacheDir)
		if cacheErr != nil {
			logger.Panic("Failed to create fsCache", zap.Error(cacheErr))
		}

		cacheHandler = localCache
	} else {
		cacheHandler = &nopCache{}
	}

	database, errDB := newStore(ctx, logger, config.DSN)
	if errDB != nil {
		logger.Fatal("Failed to connect to database", zap.Error(errDB))
	}

	pm := newProxyManager(logger)

	app := NewApp(logger, config, database, cacheHandler, pm)

	if errStart := app.Start(ctx); errStart != nil {
		logger.Error("App returned error", zap.Error(errStart))

		return
	}
}
