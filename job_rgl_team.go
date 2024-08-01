package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/leighmacdonald/bd-api/domain"
	"github.com/leighmacdonald/rgl"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/riverqueue/river"
)

type RGLTeamArgs struct {
	TeamID int `json:"team_id"`
}

func (RGLTeamArgs) Kind() string {
	return string(KindRGLTeam)
}

func (RGLTeamArgs) InsertOpts() river.InsertOpts {
	return rglInsertOpts()
}

type RGLTeamWorker struct {
	river.WorkerDefaults[RGLTeamArgs]
	database   *pgStore
	limiter    *LimiterCustom
	httpClient *http.Client
}

func (w *RGLTeamWorker) Work(ctx context.Context, job *river.Job[RGLTeamArgs]) error {
	team, errTeam := w.database.rglTeamGet(ctx, job.Args.TeamID)
	if errTeam != nil && !errors.Is(errTeam, errDatabaseNoResults) {
		return errTeam
	}

	if team.TeamID == 0 { //nolint:nestif
		w.limiter.Wait(ctx)

		fetched, errFetch := rgl.Team(ctx, NewHTTPClient(), int64(job.Args.TeamID))
		if errFetch != nil {
			return errors.Join(errFetch, errFetchTeam)
		}

		team.TeamID = fetched.TeamID
		team.SeasonID = fetched.SeasonID
		team.DivisionID = fetched.DivisionID
		team.DivisionName = fetched.DivisionName
		team.TeamLeader = steamid.New(fetched.TeamLeader)
		team.Tag = fetched.Tag
		team.TeamName = fetched.Name
		team.FinalRank = fetched.FinalRank
		team.CreatedAt = fetched.CreatedAt
		team.UpdatedAt = fetched.UpdatedAt

		record := newPlayerRecord(team.TeamLeader)
		if err := w.database.playerGetOrCreate(ctx, team.TeamLeader, &record); err != nil {
			return err
		}

		if err := w.database.rglTeamInsert(ctx, team); err != nil {
			return err
		}

		for _, player := range fetched.Players {
			memberRecord := newPlayerRecord(player.SteamID)
			if err := w.database.playerGetOrCreate(ctx, player.SteamID, &memberRecord); err != nil {
				return err
			}

			if err := w.database.rglTeamMemberInsert(ctx, domain.RGLTeamMember{
				TeamID:       team.TeamID,
				Name:         player.Name,
				IsTeamLeader: player.IsLeader,
				SteamID:      player.SteamID,
				JoinedAt:     player.JoinedAt,
				LeftAt:       &player.UpdatedOn,
			}); err != nil {
				slog.Error("Failed to ensure team member", ErrAttr(err))
			}
		}
	}

	return nil
}
