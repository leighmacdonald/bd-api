package main

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/bd-api/models"
	"github.com/pkg/errors"
)

type App struct {
	config   appConfig
	db       *pgStore
	cache    cache
	scrapers []*sbScraper
	pm       *proxyManager
	router   *gin.Engine
}

func NewApp(config appConfig, database *pgStore, cache cache, proxyManager *proxyManager) (*App, error) {
	application := &App{
		config:   config,
		db:       database,
		cache:    cache,
		pm:       proxyManager,
		router:   nil,
		scrapers: []*sbScraper{},
	}

	router, errRouter := application.createRouter()
	if errRouter != nil {
		return nil, errRouter
	}

	application.router = router

	return application, nil
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
	scrapers, errScrapers := createScrapers(a.config.CacheDir)
	if errScrapers != nil {
		return errScrapers
	}

	for _, scraper := range scrapers {
		// Attach a site_id to the scraper, so we can keep track of the scrape source
		var s models.SbSite
		if errSave := a.db.sbSiteGetOrCreate(ctx, scraper.name, &s); errSave != nil {
			return errors.Wrap(errSave, "Database error")
		}

		scraper.ID = uint32(s.SiteID)
	}

	a.scrapers = scrapers

	return nil
}
