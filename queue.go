package main

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
	"log/slog"
)

func migrateQueue(ctx context.Context, dbPool *pgxpool.Pool) error {
	tx, err := dbPool.Begin(ctx)
	if err != nil {
		panic(err)
	}
	defer tx.Rollback(ctx)

	driver := riverpgxv5.New(dbPool)

	migrator := rivermigrate.New(driver, nil)

	// Migrate to version 3. An actual call may want to omit all MigrateOpts,
	// which will default to applying all available up migrations.
	res, err := migrator.MigrateTx(ctx, tx, rivermigrate.DirectionUp, nil)
	if err != nil {
		panic(err)
	}

	for _, version := range res.Versions {
		slog.Info("Migrated river version", slog.Int("version", version.Version))
	}
}
