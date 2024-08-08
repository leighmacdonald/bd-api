package main

import (
	"context"
	"log/slog"

	"github.com/riverqueue/river"
)

type ServemeArgs struct{}

func (ServemeArgs) Kind() string {
	return string(KindServemeBan)
}

func (ServemeArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue:    string(QueueDefault),
		Priority: int(Slow),
	}
}

type ServemeWorker struct {
	river.WorkerDefaults[ServemeArgs]
	database *pgStore
}

func (w *ServemeWorker) Work(ctx context.Context, _ *river.Job[ServemeArgs]) error {
	if err := updateServeMe(ctx, w.database); err != nil {
		slog.Error("Failed to execute serveme update", ErrAttr(err))

		return err
	}

	return nil
}
