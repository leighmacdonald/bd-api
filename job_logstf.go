package main

import (
	"context"
	"log/slog"

	"github.com/riverqueue/river"
)

type LogsTFArgs struct{}

func (LogsTFArgs) Kind() string {
	return string(KindLogsTF)
}

func (LogsTFArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue:    string(QueueLogsTF),
		Priority: int(Slow),
	}
}

type LogsTFWorker struct {
	river.WorkerDefaults[LogsTFArgs]
	database *pgStore
	config   appConfig
}

func (w *LogsTFWorker) Work(ctx context.Context, _ *river.Job[LogsTFArgs]) error {
	scraper, errScraper := NewLogsTFScraper(w.database, w.config)
	if errScraper != nil {
		return errScraper
	}

	if w.config.ProxiesEnabled {
		if errProxies := attachCollectorProxies(scraper.Collector, &w.config); errProxies != nil {
			return errProxies
		}
	}

	if err := scrapeLogsTF(ctx, scraper); err != nil {
		slog.Error("Failed to scrape logs.tf", ErrAttr(err))

		return err
	}

	return nil
}
