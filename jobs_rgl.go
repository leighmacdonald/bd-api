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
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/riverqueue/river"
	"golang.org/x/time/rate"
)

const (
	rglRefillRate = 0.5
	rglBucketSize = 5
)

func NewRGLLimiter() *LimiterCustom {
	return &LimiterCustom{Limiter: rate.NewLimiter(rglRefillRate, rglBucketSize)}
}

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

type RGLBanArgs struct{}

func (RGLBanArgs) Kind() string {
	return string(KindRGLBan)
}

func (RGLBanArgs) InsertOpts() river.InsertOpts {
	return rglInsertOpts()
}

type RGLBansWorker struct {
	river.WorkerDefaults[RGLBanArgs]
	database   *pgStore
	limiter    *LimiterCustom
	httpClient *http.Client
}

func (w *RGLBansWorker) Work(ctx context.Context, _ *river.Job[RGLBanArgs]) error {
	slog.Info("Updating rgl bans")
	var (
		offset = 0
		bans   []domain.RGLBan
	)

	slog.Info("Starting RGL Bans update")

	for {
		w.limiter.Wait(ctx)

		slog.Info("Fetching RGL ban set", slog.Int("offset", offset))

		fetched, errBans := rgl.Bans(ctx, w.httpClient, 100, offset)
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

		if _, err := w.database.playerGetOrCreate(ctx, team.TeamLeader); err != nil {
			return err
		}

		if err := w.database.rglTeamInsert(ctx, team); err != nil {
			return err
		}

		for _, player := range fetched.Players {
			if _, err := w.database.playerGetOrCreate(ctx, player.SteamID); err != nil {
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
