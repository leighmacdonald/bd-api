package main

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"github.com/riverqueue/river"
	"golang.org/x/time/rate"
)

const (
	maxQueuedCount  = 100
	steamBucketSize = 200
	steamFillRate   = 1
)

var errSteamAPIResult = errors.New("failed to get data from steam api")

func NewSteamLimiter() *LimiterCustom {
	return &LimiterCustom{Limiter: rate.NewLimiter(steamFillRate, steamBucketSize)}
}

type SteamSummaryArgs struct{}

func (SteamSummaryArgs) Kind() string {
	return string(KindSteamSummary)
}

func (SteamSummaryArgs) InsertOpts() river.InsertOpts {
	return steamInsertOpts()
}

type SteamSummaryWorker struct {
	river.WorkerDefaults[SteamSummaryArgs]
	database *pgStore
	limiter  *LimiterCustom
}

func (w *SteamSummaryWorker) Timeout(_ *river.Job[SteamSummaryArgs]) time.Duration {
	return time.Second * 10
}

func (w *SteamSummaryWorker) Work(ctx context.Context, _ *river.Job[SteamSummaryArgs]) error {
	client := river.ClientFromContext[pgx.Tx](ctx)

	var expiredIDs steamid.Collection
	expiredProfiles, errProfiles := w.database.playerGetExpiredProfiles(ctx, maxQueuedCount)
	if errProfiles != nil && !errors.Is(errProfiles, errDatabaseNoResults) {
		slog.Error("Failed to fetch expired profiles", ErrAttr(errProfiles))

		return errProfiles
	}

	if len(expiredProfiles) == 0 {
		return nil
	}

	for _, profile := range expiredProfiles {
		expiredIDs = append(expiredIDs, profile.SteamID)
	}

	w.limiter.Wait(ctx)

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

	gameJobs := make([]river.InsertManyParams, len(expiredIDs))

	for index, sid := range expiredIDs {
		gameJobs[index] = river.InsertManyParams{
			Args: SteamGamesArgs{SteamID: sid},
		}
	}

	if err := w.database.insertJobsTx(ctx, client, gameJobs); err != nil {
		return err
	}

	return nil
}

type SteamBanArgs struct {
	SteamIDs steamid.Collection
}

func (SteamBanArgs) Kind() string {
	return string(KindSteamBan)
}

func (SteamBanArgs) InsertOpts() river.InsertOpts {
	return steamInsertOpts()
}

type SteamBanWorker struct {
	river.WorkerDefaults[SteamBanArgs]
	database *pgStore
	limiter  *LimiterCustom
}

func (w *SteamBanWorker) Work(ctx context.Context, job *river.Job[SteamBanArgs]) error {
	w.limiter.Wait(ctx)

	bans, errBans := steamweb.GetPlayerBans(ctx, job.Args.SteamIDs)
	if errBans != nil {
		return errors.Join(errBans, errSteamAPIResult)
	}

	for _, ban := range bans {
		var record PlayerRecord
		if errQueued := w.database.playerGetOrCreate(ctx, ban.SteamID, &record); errQueued != nil {
			continue
		}

		record.applyBans(ban)

		if errSave := w.database.playerRecordSave(ctx, &record); errSave != nil {
			return errSave
		}
	}

	return nil
}

type SteamGamesArgs struct {
	SteamID steamid.SteamID `json:"steam_id"`
}

func (SteamGamesArgs) Kind() string {
	return string(KindSteamGames)
}

func (SteamGamesArgs) InsertOpts() river.InsertOpts {
	return steamInsertOpts()
}

type SteamGamesWorker struct {
	river.WorkerDefaults[SteamGamesArgs]
	database *pgStore
	limiter  *LimiterCustom
}

func (w *SteamGamesWorker) Work(ctx context.Context, job *river.Job[SteamGamesArgs]) error {
	// https://wiki.teamfortress.com/wiki/WebAPI/GetOwnedGames
	slog.Debug("Updating games", slog.String("steam_id", job.Args.SteamID.String()))

	// Back off api calls a bit so we don't error out since this endpoint does not support querying more
	// than a single steamid at a time.
	// We should be able to do a request every 1.1~ seconds non-stop with a single IP/API Key.
	// 100000 (req allowed / 24hr) / 86400 (secs/day) = ~1.15 req/sec
	w.limiter.Wait(ctx)

	if _, err := updateOwnedGames(ctx, w.database, job.Args.SteamID); err != nil {
		slog.Error("Failed to update owned games", slog.String("sid64", job.Args.SteamID.String()), ErrAttr(err))

		return err
	}

	return nil
}
