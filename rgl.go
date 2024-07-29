package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/leighmacdonald/rgl"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

var (
	errFetchTeam   = errors.New("failed to fetch rgl team via api")
	errFetchSeason = errors.New("failed to fetch rgl season via api")
	errFetchBans   = errors.New("failed to fetch rgl bans")
)

type RGLSeason struct {
	SeasonID           int       `json:"season_id"`
	Name               string    `json:"name"`
	Maps               []string  `json:"maps"`
	FormatName         string    `json:"format_name"`
	RegionName         string    `json:"region_name"`
	ParticipatingTeams []int     `json:"participating_teams"`
	CreatedOn          time.Time `json:"created_on"`
}

type RGLTeam struct {
	TeamID       int             `json:"team_id,omitempty"`
	SeasonID     int             `json:"season_id,omitempty"`
	DivisionID   int             `json:"division_id,omitempty"`
	DivisionName string          `json:"division_name,omitempty"`
	TeamLeader   steamid.SteamID `json:"team_leader"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	Tag          string          `json:"tag,omitempty"`
	TeamName     string          `json:"team_name,omitempty"`
	FinalRank    int             `json:"final_rank,omitempty"`
	// TeamStatus   string
	// TeamReady    bool
}

type RGLTeamMember struct {
	TeamID       int             `json:"team_id"`
	Name         string          `json:"name"`
	IsTeamLeader bool            `json:"is_team_leader"`
	SteamID      steamid.SteamID `json:"steam_id"`
	JoinedAt     time.Time       `json:"joined_at"`
	LeftAt       *time.Time      `json:"left_at"`
}

type RGLBan struct {
	SteamID   steamid.SteamID `json:"steam_id"`
	Alias     string          `json:"alias"`
	ExpiresAt time.Time       `json:"expires_at"`
	CreatedAt time.Time       `json:"created_at"`
	Reason    string          `json:"reason"`
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
		r.waitTime *= 2
	}

	time.Sleep(r.waitTime)
}

// scrapeRGL handles fetching all of the RGL data.
// It operates in the following order:
//
// - Fetch bans
// - Fetch season
// - Fetch season teams
// - Fetch season team members
// - Fetch season matches?
func (r *RGLScraper) scrapeRGL(ctx context.Context) {
	var (
		curID    = 1
		waitTime = time.Second
		maxErr   = 100
		curErr   = 0
	)

	if err := r.updateBans(ctx); err != nil {
		slog.Error("Failed to update bans", ErrAttr(err))
	}

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

		curID++
	}

	slog.Info("Max errors reached. Stopping RGL update.")
}

func (r *RGLScraper) getRGLTeam(ctx context.Context, teamID int) (RGLTeam, error) {
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

			if err := r.database.rglTeamMemberInsert(ctx, RGLTeamMember{
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

func (r *RGLScraper) getRGLSeason(ctx context.Context, seasonID int) (RGLSeason, error) {
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
		season.CreatedOn = time.Now()

		if err := r.database.rglSeasonInsert(ctx, season); err != nil {
			return season, err
		}
	}

	return season, nil
}

func (r *RGLScraper) updateBans(ctx context.Context) error {
	var (
		offset = 0
		bans   []RGLBan
	)

	slog.Info("Starting RGL Bans update")

	for {
		slog.Info("Fetching RGL ban set", slog.Int("offset", offset))

		fetched, errBans := rgl.Bans(ctx, r.client, 100, offset)
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

			bans = append(bans, RGLBan{
				SteamID:   sid,
				Alias:     ban.Alias,
				ExpiresAt: ban.ExpiresAt,
				CreatedAt: ban.CreatedAt,
				Reason:    ban.Reason,
			})
		}

		r.waiter(nil)

		offset += 100
	}

	if err := r.database.rglBansReplace(ctx, bans); err != nil {
		return err
	}

	slog.Info("Updated RGL bans successfully", slog.Int("count", len(bans)))

	return nil
}
