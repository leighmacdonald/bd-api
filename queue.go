package main

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
)

var errQueueSetup = errors.New("could not setup job queue")

func setupQueue(ctx context.Context, dbPool *pgxpool.Pool) error {
	driver := riverpgxv5.New(dbPool)

	migrator := rivermigrate.New[pgx.Tx](driver, nil)

	res, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, nil)
	if err != nil {
		panic(err)
	}

	for _, ver := range res.Versions {
		slog.Info("Migrated river version", slog.Int("version", ver.Version))
	}

	return nil
}

func createJobWorkers(database *pgStore) *river.Workers {
	workers := river.NewWorkers()
	river.AddWorker[RGLBanArgs](workers, &RGLBansWorker{
		database: database,
		limiter:  NewRGLLimiter(),
	})
	river.AddWorker[ProfileArgs](workers, &ProfileWorker{})

	return workers
}

func CreateJobClient(dbPool *pgxpool.Pool, maxWorkers int, workers *river.Workers) (*river.Client[pgx.Tx], error) {
	riverClient, err := river.NewClient[pgx.Tx](riverpgxv5.New(dbPool), &river.Config{
		Logger:     slog.Default(),
		JobTimeout: time.Minute * 5,
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: maxWorkers},
		},
		Workers: workers,
		PeriodicJobs: []*river.PeriodicJob{
			river.NewPeriodicJob(
				river.PeriodicInterval(6*time.Hour),
				func() (river.JobArgs, *river.InsertOpts) {
					return RGLBanArgs{}, nil
				},
				&river.PeriodicJobOpts{RunOnStart: true}),
		},
		MaxAttempts: 3,
	})
	if err != nil {
		return nil, errors.Join(err, errQueueSetup)
	}

	return riverClient, nil
}

func startJobClient(ctx context.Context, pool *pgxpool.Pool, maxWorkers int, workers *river.Workers) (*river.Client[pgx.Tx], error) {
	riverClient, err := CreateJobClient(pool, maxWorkers, workers)
	if err != nil {
		return nil, err
	}

	if errClientStart := riverClient.Start(ctx); errClientStart != nil {
		return nil, errors.Join(errClientStart, errQueueSetup)
	}

	return riverClient, nil
}
