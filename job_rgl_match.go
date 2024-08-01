package main

import (
	"context"
	"errors"
	"net/http"

	"github.com/leighmacdonald/rgl"
	"github.com/riverqueue/river"
)

type RGLMatchArgs struct {
	MatchID int `json:"match_id"`
}

func (RGLMatchArgs) Kind() string {
	return string(KindRGLMatch)
}

func (RGLMatchArgs) InsertOpts() river.InsertOpts {
	return rglInsertOpts()
}

type RGLMatchWorker struct {
	river.WorkerDefaults[RGLMatchArgs]
	database   *pgStore
	limiter    *LimiterCustom
	httpClient *http.Client
}

func (w *RGLMatchWorker) Work(ctx context.Context, job *river.Job[RGLMatchArgs]) error {
	match, errMatch := w.database.rglMatchGet(ctx, job.Args.MatchID)
	if errMatch != nil && !errors.Is(errMatch, errDatabaseNoResults) {
		return errMatch
	}

	if match.MatchID == 0 {
		w.limiter.Wait(ctx)

		fetched, err := rgl.Match(ctx, NewHTTPClient(), int64(job.Args.MatchID))
		if err != nil {
			return errors.Join(err, errFetchMatch)
		}

		if len(fetched.Teams) < 2 {
			return errors.Join(err, errFetchMatchInvalid)
		}

		match.MatchID = fetched.MatchID
		match.MatchName = fetched.MatchName
		match.MatchDate = fetched.MatchDate
		match.DivisionName = fetched.DivisionName
		match.DivisionID = fetched.DivisionID
		match.RegionID = fetched.RegionID
		match.MatchDate = fetched.MatchDate
		match.IsForfeit = fetched.IsForfeit
		match.Winner = fetched.Winner
		match.SeasonName = fetched.SeasonName
		match.SeasonID = fetched.SeasonID
		match.TeamIDA = fetched.Teams[0].TeamID
		match.PointsA = fetched.Teams[0].Points
		match.TeamIDB = fetched.Teams[1].TeamID
		match.PointsB = fetched.Teams[1].Points

		if errInsert := w.database.rglMatchInsert(ctx, match); errInsert != nil {
			return errInsert
		}
	}

	return nil
}
