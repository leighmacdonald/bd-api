package main

import (
	"context"
	"log/slog"

	"github.com/riverqueue/river"
)

type BDListArgs struct{}

func (BDListArgs) Kind() string {
	return string(KindBDLists)
}

func (BDListArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue:    string(QueueDefault),
		Priority: int(Slow),
	}
}

type BDListWorker struct {
	river.WorkerDefaults[BDListArgs]
	database *pgStore
}

func (w *BDListWorker) Work(ctx context.Context, _ *river.Job[BDListArgs]) error {
	if err := doListUpdate(ctx, w.database); err != nil {
		slog.Error("Failed to update bd lists", ErrAttr(err))

		return err
	}

	return nil
}
