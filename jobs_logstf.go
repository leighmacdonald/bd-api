package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/riverqueue/river"
)

type LogsTFArgs struct{}

func (LogsTFArgs) Kind() string {
	return string(KindLogsTF)
}

func (LogsTFArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue:      string(QueueLogsTF),
		Priority:   int(Slow),
		UniqueOpts: river.UniqueOpts{ByPeriod: time.Hour},
	}
}

type LogsTFWorker struct {
	river.WorkerDefaults[LogsTFArgs]
	database *pgStore
	config   appConfig
}

func (w *LogsTFWorker) Timeout(_ *river.Job[LogsTFArgs]) time.Duration {
	return time.Hour * 6
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
