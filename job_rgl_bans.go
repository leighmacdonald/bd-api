package main

import (
	"context"
	"errors"
	"log/slog"

	"github.com/leighmacdonald/bd-api/domain"
	"github.com/leighmacdonald/rgl"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/riverqueue/river"
	"golang.org/x/time/rate"
)

type RGLBanArgs struct{}

func (RGLBanArgs) Kind() string {
	return "rgl_bans"
}

type RGLBansWorker struct {
	river.WorkerDefaults[RGLBanArgs]
	database *pgStore
	limiter  *rate.Limiter
}

func (w *RGLBansWorker) Work(ctx context.Context, _ *river.Job[RGLBanArgs]) error {
	slog.Info("Updating rgl bans")
	var (
		offset = 0
		bans   []domain.RGLBan
	)

	slog.Info("Starting RGL Bans update")

	client := NewHTTPClient()

	for {
		if err := w.limiter.Wait(ctx); err != nil {
			slog.Error("Failed to wait for rgl limiter", ErrAttr(err))
		}

		slog.Info("Fetching RGL ban set", slog.Int("offset", offset))

		fetched, errBans := rgl.Bans(ctx, client, 100, offset)
		if errBans != nil {
			return errors.Join(errBans, errFetchBans)
		}

		if len(fetched) == 0 {
			break
		}

		for _, ban := range fetched {
			sid := steamid.New(ban.SteamID)
			if !sid.Valid() {
				// A couple entries seem to have a 0 value for SID
				continue
			}

			bans = append(bans, domain.RGLBan{
				SteamID:   sid,
				Alias:     ban.Alias,
				ExpiresAt: ban.ExpiresAt,
				CreatedAt: ban.CreatedAt,
				Reason:    ban.Reason,
			})
		}

		offset += 100
	}

	if err := w.database.rglBansReplace(ctx, bans); err != nil {
		return err
	}

	slog.Info("Updated RGL bans successfully", slog.Int("count", len(bans)))

	return nil
}
