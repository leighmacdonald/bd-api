package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/riverqueue/river"
)

type SourcebansArgs struct{}

func (SourcebansArgs) Kind() string {
	return string(KindSourcebans)
}

func (SourcebansArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue:      string(QueueSourcebans),
		Priority:   int(Slow),
		UniqueOpts: river.UniqueOpts{ByPeriod: time.Hour * 6},
	}
}

type SourcebansWorker struct {
	river.WorkerDefaults[SourcebansArgs]
	database *pgStore
	config   appConfig
}

func (w *SourcebansWorker) Timeout(_ *river.Job[SourcebansArgs]) time.Duration {
	return time.Hour * 6
}

func (w *SourcebansWorker) Work(ctx context.Context, _ *river.Job[SourcebansArgs]) error {
	if err := runSourcebansScraper(ctx, w.database, w.config); err != nil {
		slog.Error("Failed to update sourcebans list", ErrAttr(err))

		return err
	}

	return nil
}
