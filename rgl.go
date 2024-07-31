package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/leighmacdonald/bd-api/domain"
	"github.com/leighmacdonald/rgl"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"golang.org/x/time/rate"
)

var (
	errFetchMatch        = errors.New("failed to fetch rgl match via api")
	errFetchMatchInvalid = errors.New("fetched invalid rgl match via api")
	errFetchTeam         = errors.New("failed to fetch rgl team via api")
	errFetchSeason       = errors.New("failed to fetch rgl season via api")
	errFetchBans         = errors.New("failed to fetch rgl bans")
)

const (
	rglRefillRate = 0.5
	rglBucketSize = 5
)

func NewRGLLimiter() *rate.Limiter {
	return rate.NewLimiter(rglRefillRate, rglBucketSize)
}

func NewRGLScraper(database *pgStore) RGLScraper {
	return RGLScraper{
		database: database,
		curID:    0,
		waitTime: time.Second,
		client:   NewHTTPClient(),
	}
}

type RGLScraper struct {
	database *pgStore
	curID    int
	waitTime time.Duration
	client   *http.Client
}

func (r *RGLScraper) start(ctx context.Context) {
	scraperInterval := time.Hour
	scraperTimer := time.NewTimer(scraperInterval)

	r.scrapeRGL(ctx)

	for {
		select {
		case <-scraperTimer.C:
			r.scrapeRGL(ctx)
			scraperTimer.Reset(scraperInterval)
		case <-ctx.Done():
			return
		}
	}
}

func (r *RGLScraper) waiter(err error) {
	if err != nil {
		r.waitTime = min(r.waitTime*2, time.Second*30)
	}

	time.Sleep(r.waitTime)
}

// scrapeRGL handles fetching all the RGL data.
// It operates in the following order:
//
// - Fetch bans
// - Fetch season
// - Fetch season teams
// - Fetch season team members
// - Fetch season matches.
func (r *RGLScraper) scrapeRGL(ctx context.Context) {
	var (
		curID    = 1
		waitTime = time.Second
		maxErr   = 100
		curErr   = 0
	)

	for curErr < maxErr {
		season, errSeason := r.getRGLSeason(ctx, curID)
		if errSeason != nil {
			if errSeason.Error() == "invalid status code: 404 Not Found" {
				curID++
				maxErr++
				r.waiter(errSeason)

				continue
			}

			if errors.Is(errSeason, rgl.ErrRateLimit) {
				slog.Error("Failed to fetch season (rate limited)", slog.Int("season", curID), slog.Duration("waitTime", waitTime), ErrAttr(errSeason))
				r.waiter(errSeason)

				continue
			}

			slog.Error("Unhandled error", ErrAttr(errSeason))
			curID++
			curErr++

			continue
		}

		r.waiter(nil)

		slog.Info("Got RGL season", slog.Int("season_id", season.SeasonID))

		for _, teamID := range season.ParticipatingTeams {
			team, errTeam := r.getRGLTeam(ctx, teamID)
			if errTeam != nil {
				slog.Error("Failed to fetch team", ErrAttr(errTeam))
				r.waiter(errTeam)

				continue
			}

			slog.Info("Got team", slog.String("name", team.TeamName))
		}

		for _, matchID := range season.Matches {
			match, errMatch := r.getRGLMatch(ctx, matchID)
			if errMatch != nil {
				if errors.Is(errMatch, errDatabaseUnique) {
					slog.Warn("Match team does not exist", ErrAttr(errMatch))

					continue
				}

				slog.Error("Failed to fetch match", ErrAttr(errMatch))

				continue
			}

			slog.Info("Got RGL match", slog.String("name", match.MatchName), slog.String("season", match.SeasonName))
		}

		curID++
	}

	slog.Info("Max errors reached. Stopping RGL update.")
}

func (r *RGLScraper) getRGLTeam(ctx context.Context, teamID int) (domain.RGLTeam, error) {
	team, errTeam := r.database.rglTeamGet(ctx, teamID)
	if errTeam != nil && !errors.Is(errTeam, errDatabaseNoResults) {
		return team, errTeam
	}

	if team.TeamID == 0 { //nolint:nestif
		fetched, errFetch := rgl.Team(ctx, NewHTTPClient(), int64(teamID))
		if errFetch != nil {
			r.waiter(errFetch)

			return team, errors.Join(errFetch, errFetchTeam)
		}

		r.waiter(nil)

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
		if err := r.database.playerGetOrCreate(ctx, team.TeamLeader, &record); err != nil {
			return team, err
		}

		if err := r.database.rglTeamInsert(ctx, team); err != nil {
			return team, err
		}

		for _, player := range fetched.Players {
			memberRecord := newPlayerRecord(player.SteamID)
			if err := r.database.playerGetOrCreate(ctx, player.SteamID, &memberRecord); err != nil {
				return team, err
			}

			if err := r.database.rglTeamMemberInsert(ctx, domain.RGLTeamMember{
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

	return team, nil
}

func (r *RGLScraper) getRGLSeason(ctx context.Context, seasonID int) (domain.RGLSeason, error) {
	season, errSeason := r.database.rglSeasonGet(ctx, seasonID)
	if errSeason != nil && !errors.Is(errSeason, errDatabaseNoResults) {
		return season, errSeason
	}

	if season.SeasonID == 0 {
		fetchedSeason, errFetch := rgl.Season(ctx, r.client, int64(seasonID))
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

		if err := r.database.rglSeasonInsert(ctx, season); err != nil {
			return season, err
		}
	}

	return season, nil
}

func (r *RGLScraper) getRGLMatch(ctx context.Context, matchID int) (domain.RGLMatch, error) {
	match, errMatch := r.database.rglMatchGet(ctx, matchID)
	if errMatch != nil && !errors.Is(errMatch, errDatabaseNoResults) {
		return match, errMatch
	}

	if match.MatchID == 0 {
		r.waiter(nil)

		fetched, err := rgl.Match(ctx, NewHTTPClient(), int64(matchID))
		if err != nil {
			return match, errors.Join(err, errFetchMatch)
		}

		if len(fetched.Teams) < 2 {
			return match, errors.Join(err, errFetchMatchInvalid)
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

		if errInsert := r.database.rglMatchInsert(ctx, match); errInsert != nil {
			return match, errInsert
		}
	}

	return match, nil
}
