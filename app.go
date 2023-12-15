package main

import (
	"context"
	"github.com/leighmacdonald/etf2l"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/bd-api/models"
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
	etf2l    *etf2l.Client
}

func NewApp(logger *zap.Logger, config appConfig, database *pgStore, cache cache, proxyManager *proxyManager) *App {
	application := &App{
		config:   config,
		log:      logger,
		db:       database,
		cache:    cache,
		pm:       proxyManager,
		router:   nil,
		scrapers: []*sbScraper{},
		etf2l:    etf2l.New(),
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
		var s models.SbSite
		if errSave := sbSiteGetOrCreate(ctx, a.db, scraper.name, &s); errSave != nil {
			return errors.Wrap(errSave, "Database error")
		}

		scraper.ID = uint32(s.SiteID)
	}

	a.scrapers = scrapers

	return nil
}

func (a *App) updatePlayer(ctx context.Context, steamID steamid.SID64) error {
	lCtx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	startTime := time.Now()

	// errgroup is not used since we want to update as much as possible even if some fail
	wg := &sync.WaitGroup{}
	wg.Add(2)

	go func() {
		wg.Done()
		if err := a.updateETF2LPlayer(lCtx, steamID); err != nil {
			a.log.Warn("Failed to updated ETF2L profile", zap.Int64("steam_id", steamID.Int64()), zap.Error(err))

			return
		}
	}()

	go func() {
		wg.Done()
		if err := a.updateRGLPlayer(lCtx, steamID); err != nil {
			a.log.Warn("Failed to updated RGL profile", zap.Int64("steam_id", steamID.Int64()), zap.Error(err))
			return
		}
	}()

	wg.Wait()

	a.log.Debug("Player update finished", zap.Int64("steam_id", steamID.Int64()), zap.Duration("duration", time.Since(startTime)))

	return nil
}

func (a *App) updateETF2LPlayer(ctx context.Context, steamID steamid.SID64) error {
	player, err := a.etf2l.Player(ctx, steamID.String())
	if err != nil {
		return err
	}

	return nil
}

func (a *App) updateRGLPlayer(ctx context.Context, steamID steamid.SID64) error {
	return nil
}
