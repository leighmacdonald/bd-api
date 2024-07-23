package main

import (
	"errors"
	"github.com/gocolly/colly"
	"github.com/gocolly/colly/extensions"
	"github.com/gocolly/colly/queue"
	"log/slog"
	"path/filepath"
	"strings"
	"time"
)

type SiteScraper struct {
	*colly.Collector
	log   *slog.Logger
	queue *queue.Queue
	db    *pgStore
	delay time.Duration
}

func NewScraper(database *pgStore, config appConfig, domain string) (*SiteScraper, error) {
	name := strings.ReplaceAll(domain, ".", "")
	logger := slog.With("name", name)
	debugLogger := scrapeLogger{logger: logger} //nolint:exhaustruct

	reqQueue, errQueue := queue.New(6, &queue.InMemoryQueueStorage{})
	if errQueue != nil {
		return nil, errors.Join(errQueue, errScrapeQueueInit)
	}

	collector := colly.NewCollector(
		colly.UserAgent("bd-api"),
		colly.CacheDir(filepath.Join(config.CacheDir, name)),
		colly.Debugger(&debugLogger),
		colly.AllowedDomains(domain),
	)

	extensions.RandomUserAgent(collector)

	scraper := SiteScraper{
		Collector: collector,
		log:       logger,
		queue:     reqQueue,
		db:        database,
	}

	scraper.SetRequestTimeout(requestTimeout)
	scraper.OnRequest(func(r *colly.Request) {
		logger.Debug("Visiting", slog.String("url", r.URL.String()))
	})

	scraper.delay = time.Duration(config.ScrapeDelay) * time.Millisecond

	parallelism := 1
	if config.ProxiesEnabled {
		parallelism = len(config.Proxies)
	}

	if errLimit := scraper.Limit(&colly.LimitRule{ //nolint:exhaustruct
		DomainGlob:  "*" + domain,
		Delay:       scraper.delay,
		Parallelism: parallelism,
	}); errLimit != nil {
		return nil, errors.Join(errLimit, errScrapeLimit)
	}

	return &scraper, nil
}
