package main

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type App struct {
	config   appConfig
	db       *pgStore
	log      *zap.Logger
	cache    cache
	scrapers []*sbScraper
	pm       *proxyManager
	router   *gin.Engine
}

func NewApp(logger *zap.Logger, config appConfig, database *pgStore, cache cache, proxyManager *proxyManager) *App {
	application := &App{
		config:   config,
		log:      logger.Named("api"),
		db:       database,
		cache:    cache,
		pm:       proxyManager,
		router:   nil,
		scrapers: []*sbScraper{},
	}

	router, errRouter := application.createRouter()
	if errRouter != nil {
		logger.Fatal("Failed to create router", zap.Error(errRouter))
	}

	application.router = router

	return application
}

func (a *App) Start(ctx context.Context) error {
	if a.config.SourcebansScraperEnabled {
		if errInitScrapers := a.initScrapers(ctx); errInitScrapers != nil {
			return errInitScrapers
		}

		go a.startScrapers(ctx)
	}

	go a.profileUpdater(ctx)

	return a.startAPI(ctx, a.config.ListenAddr)
}

func (a *App) initScrapers(ctx context.Context) error {
	scrapers, errScrapers := createScrapers(a.log, a.config.CacheDir)
	if errScrapers != nil {
		return errScrapers
	}

	for _, scraper := range scrapers {
		// Attach a site_id to the scraper, so we can keep track of the scrape source
		var s sbSite
		if errSave := a.db.sbSiteGetOrCreate(ctx, scraper.name, &s); errSave != nil {
			return errors.Wrap(errSave, "Database error")
		}

		scraper.ID = uint32(s.SiteID)
	}

	a.scrapers = scrapers

	return nil
}
