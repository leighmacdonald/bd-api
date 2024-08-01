package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/leighmacdonald/bd-api/domain"
	"github.com/leighmacdonald/rgl"
	"github.com/riverqueue/river"
)

type RGLSeasonArgs struct {
	StartSeasonID int
}

func (RGLSeasonArgs) Kind() string {
	return string(KindRGLSeason)
}

func (RGLSeasonArgs) InsertOpts() river.InsertOpts {
	return rglInsertOpts()
}

type RGLSeasonWorker struct {
	river.WorkerDefaults[RGLSeasonArgs]
	database   *pgStore
	limiter    *LimiterCustom
	httpClient *http.Client
}

func (w *RGLSeasonWorker) Work(ctx context.Context, job *river.Job[RGLSeasonArgs]) error {
	var (
		client   = river.ClientFromContext[pgx.Tx](ctx)
		curID    = job.Args.StartSeasonID
		waitTime = time.Second
		maxErr   = 100
		curErr   = 0
	)

	for curErr < maxErr {
		season, errSeason := w.updateSeason(ctx, w.database, curID)
		if errSeason != nil {
			if errSeason.Error() == "invalid status code: 404 Not Found" {
				curID++
				maxErr++

				continue
			}

			if errors.Is(errSeason, rgl.ErrRateLimit) {
				slog.Error("Failed to fetch season (rate limited)", slog.Int("season", curID), slog.Duration("waitTime", waitTime), ErrAttr(errSeason))

				continue
			}

			slog.Error("Unhandled error", ErrAttr(errSeason))
			curID++
			curErr++

			continue
		}

		slog.Info("Got RGL season", slog.Int("season_id", season.SeasonID))

		var newJobs []river.InsertManyParams
		for _, teamID := range season.ParticipatingTeams {
			newJobs = append(newJobs, river.InsertManyParams{Args: RGLTeamArgs{TeamID: teamID}})
		}

		for _, matchID := range season.Matches {
			newJobs = append(newJobs, river.InsertManyParams{Args: RGLMatchArgs{MatchID: matchID}})
		}

		if err := w.database.insertJobsTx(ctx, client, newJobs); err != nil {
			return err
		}

		curID++
	}

	slog.Info("Max errors reached. Stopping RGL update.")

	return nil
}

func (w *RGLSeasonWorker) updateSeason(ctx context.Context, database *pgStore, seasonID int) (domain.RGLSeason, error) {
	season, errSeason := database.rglSeasonGet(ctx, seasonID)
	if errSeason != nil && !errors.Is(errSeason, errDatabaseNoResults) {
		return season, errSeason
	}

	if season.SeasonID == 0 {
		w.limiter.Wait(ctx)

		fetchedSeason, errFetch := rgl.Season(ctx, w.httpClient, int64(seasonID))
		if errFetch != nil {
			return season, errors.Join(errFetch, errFetchSeason)
		}

		season.SeasonID = seasonID
		season.Name = fetchedSeason.Name
		season.Maps = fetchedSeason.Maps
		season.RegionName = fetchedSeason.RegionName
		season.FormatName = fetchedSeason.FormatName
		season.ParticipatingTeams = fetchedSeason.ParticipatingTeams
		season.Matches = fetchedSeason.MatchesPlayedDuringSeason
		season.CreatedOn = time.Now()

		if err := database.rglSeasonInsert(ctx, season); err != nil {
			return season, err
		}
	}

	return season, nil
}
