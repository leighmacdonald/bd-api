package main

import (
	"context"

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
	return updateServeMe(ctx, w.database)
}
