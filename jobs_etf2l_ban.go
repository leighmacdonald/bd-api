package main

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/leighmacdonald/etf2l"
	"github.com/riverqueue/river"
)

type ETF2LBanArgs struct{}

func (ETF2LBanArgs) Kind() string {
	return string(KindETF2LBan)
}

func (ETF2LBanArgs) InsertOpts() river.InsertOpts {
	return etf2lInsertOpts()
}

type ETF2LBanWorker struct {
	river.WorkerDefaults[ETF2LBanArgs]
	database   *pgStore
	client     *etf2l.Client
	httpClient *http.Client
}

func (w *ETF2LBanWorker) Work(ctx context.Context, _ *river.Job[ETF2LBanArgs]) error {
	if err := updateETF2LBans(ctx, w.database, w.client, w.httpClient); err != nil {
		slog.Error("Failed to updated etf2l bans", ErrAttr(err))

		return err
	}

	return nil
}
