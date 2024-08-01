package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"github.com/riverqueue/river"
)

var errSteamAPIResult = errors.New("failed to get data from steam api")

type SteamSummaryArgs struct{}

func (SteamSummaryArgs) Kind() string {
	return string(KindSteamSummary)
}

func (SteamSummaryArgs) InsertOpts() river.InsertOpts {
	return steamInsertOpts()
}

type SteamSummaryWorker struct {
	river.WorkerDefaults[SteamSummaryArgs]
	database   *pgStore
	httpClient *http.Client
}

func (w *SteamSummaryWorker) Work(ctx context.Context, _ *river.Job[SteamSummaryArgs]) error {
	var (
		client         = river.ClientFromContext[pgx.Tx](ctx)
		updateQueue    steamid.Collection
		maxQueuedCount = 100
	)

	var expiredIDs steamid.Collection
	expiredProfiles, errProfiles := w.database.playerGetExpiredProfiles(ctx, maxQueuedCount)
	if errProfiles != nil && !errors.Is(errProfiles, errDatabaseNoResults) {
		slog.Error("Failed to fetch expired profiles", ErrAttr(errProfiles))

		return errProfiles
	}

	additional := 0

	for len(expiredProfiles) < maxQueuedCount {
		for _, sid64 := range updateQueue {
			var pr PlayerRecord
			if errQueued := w.database.playerGetOrCreate(ctx, sid64, &pr); errQueued != nil {
				continue
			}

			expiredProfiles = append(expiredProfiles, pr)
			additional++
		}
	}

	if len(expiredProfiles) == 0 {
		return nil
	}

	for _, profile := range expiredProfiles {
		expiredIDs = append(expiredIDs, profile.SteamID)
	}

	summaries, errSum := steamweb.PlayerSummaries(ctx, expiredIDs)
	if errSum != nil {
		return errors.Join(errSum, errSteamAPIResult)
	}

	for _, profile := range expiredProfiles {
		prof := profile
		for _, sum := range summaries {
			if sum.SteamID.Int64() == prof.SteamID.Int64() {
				prof.applySummary(sum)

				break
			}
		}

		if errSave := w.database.playerRecordSave(ctx, &prof); errSave != nil {
			slog.Error("Failed to update profile", slog.Int64("sid", prof.SteamID.Int64()), ErrAttr(errSave))
		}
	}

	if err := w.database.insertJobTx(ctx, client, SteamBanArgs{SteamIDs: expiredIDs}, nil); err != nil {
		return err
	}

	// var gameJobs []river.InsertManyParams
	// for _, sid := range expiredIDs {
	//	gameJobs = append(gameJobs, river.InsertManyParams{
	//		Args: SteamGamesArgs{SteamID: sid},
	//	})
	// }
	//
	// if err := w.database.insertJobsTx(ctx, client, gameJobs); err != nil {
	//	return err
	// }

	return nil
}
