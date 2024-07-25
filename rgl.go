package main

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/bd-api/domain"
	"github.com/leighmacdonald/rgl"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

var (
	errFetchTeam   = errors.New("failed to fetch rgl team via api")
	errFetchSeason = errors.New("failed to fetch rgl season via api")
)

// Current issues:
// - Sometime api just fails on the first attempt
// - Empty value.
func getRGL(ctx context.Context, log *slog.Logger, sid64 steamid.SteamID) ([]domain.Season, error) {
	startTime := time.Now()
	client := NewHTTPClient()

	_, errProfile := rgl.Profile(ctx, client, sid64)
	if errProfile != nil {
		return nil, errors.Join(errProfile, errRequestPerform)
	}

	teams, errTeams := rgl.ProfileTeams(ctx, client, sid64)
	if errTeams != nil {
		return nil, errRequestPerform
	}

	seasons := make([]domain.Season, len(teams))

	for index, team := range teams {
		seasonStartTime := time.Now()

		var season domain.Season
		seasonInfo, errSeason := rgl.Season(ctx, client, team.SeasonID)

		if errSeason != nil {
			return nil, errRequestPerform
		}

		season.League = "rgl"
		season.Division = seasonInfo.Name
		season.DivisionInt = parseRGLDivision(team.DivisionName)
		season.TeamName = team.TeamName

		lowerName := strings.ToLower(seasonInfo.Name)

		if seasonInfo.FormatName == "" {
			switch {
			case strings.Contains(lowerName, "sixes"):

				seasonInfo.FormatName = "Sixes"
			case strings.Contains(lowerName, "prolander"):

				seasonInfo.FormatName = "Prolander"
			case strings.Contains(lowerName, "hl season"):

				seasonInfo.FormatName = "HL"
			case strings.Contains(lowerName, "p7 season"):

				seasonInfo.FormatName = "Prolander"
			}
		}

		season.Format = seasonInfo.FormatName
		seasons[index] = season

		log.Info("RGL season fetched", slog.Duration("duration", time.Since(seasonStartTime)))
	}

	log.Info("RGL Completed", slog.Duration("duration", time.Since(startTime)))

	return seasons, nil
}

func parseRGLDivision(div string) domain.Division {
	switch div {
	case "RGL-Invite":
		fallthrough
	case "Invite":
		return domain.RGLRankInvite
	case "RGL-Div-1":
		return domain.RGLRankDiv1
	case "RGL-Div-2":
		return domain.RGLRankDiv2
	case "RGL-Main":
		fallthrough
	case "Main":
		return domain.RGLRankMain
	case "RGL-Advanced":
		fallthrough
	case "Advanced-1":
		fallthrough
	case "Advanced":
		return domain.RGLRankAdvanced
	case "RGL-Intermediate":
		fallthrough
	case "Intermediate":
		return domain.RGLRankIntermediate
	case "RGL-Challenger":
		return domain.RGLRankIntermediate
	case "Open":
		return domain.RGLRankOpen
	case "Amateur":
		return domain.RGLRankAmateur
	case "Fresh Meat":
		return domain.RGLRankFreshMeat
	default:
		return domain.RGLRankNone
	}
}

func startRGLScraper(ctx context.Context, database *pgStore) {
	scraperInterval := time.Hour
	scraperTimer := time.NewTimer(scraperInterval)

	scrapeRGL(ctx, database)

	for {
		select {
		case <-scraperTimer.C:
			scrapeRGL(ctx, database)
			scraperTimer.Reset(scraperInterval)
		case <-ctx.Done():
			return
		}
	}
}

// scrapeRGL handles fetching all of the RGL data.
// It operates in the following order:
//
// - Fetch bans
// - Fetch season
// - Fetch season teams
// - Fetch season team members
// - Fetch season matches?
func scrapeRGL(ctx context.Context, database *pgStore) {
	var (
		curID    = 1
		waitTime = time.Second
		maxErr   = 100
		curErr   = 0
	)

	waiter := func(err error) {
		if err != nil {
			waitTime *= 2
		}
		time.Sleep(waitTime)
	}

	for curErr < maxErr {
		season, errSeason := getRGLSeason(ctx, database, curID)
		if errSeason != nil {
			if errSeason.Error() == "invalid status code: 404 Not Found" {
				curID++
				maxErr++
				waiter(errSeason)

				continue
			}

			if errors.Is(errSeason, rgl.ErrRateLimit) {
				slog.Error("Failed to fetch season (rate limited)", slog.Int("season", curID), slog.Duration("waitTime", waitTime), ErrAttr(errSeason))
				waiter(errSeason)

				continue
			}

			slog.Error("Unhandled error", ErrAttr(errSeason))
			curID++
			curErr++

			continue
		}

		waiter(nil)

		slog.Info("Got RGL season", slog.Int("season_id", season.SeasonID))

		for _, teamID := range season.ParticipatingTeams {
			team, errTeam := getRGLTeam(ctx, database, teamID)
			if errTeam != nil {
				slog.Error("Failed to fetch team", ErrAttr(errTeam))
				waiter(errTeam)

				continue
			}

			slog.Info("Got team", slog.String("name", team.TeamName))
		}

		curID++
	}
}

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
	TeamID       int
	SeasonID     int
	DivisionID   int
	DivisionName string
	TeamLeader   steamid.SteamID
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Tag          string
	TeamName     string
	FinalRank    int
	// TeamStatus   string
	// TeamReady    bool
}

func getRGLTeam(ctx context.Context, database *pgStore, teamID int) (RGLTeam, error) {
	team, errTeam := rglTeamGet(ctx, database, teamID)
	if errTeam != nil && !errors.Is(errTeam, errDatabaseNoResults) {
		return team, errTeam
	}

	if team.TeamID == 0 {
		fetched, errFetch := rgl.Team(ctx, NewHTTPClient(), int64(teamID))
		if errFetch != nil {
			return team, errors.Join(errFetch, errFetchTeam)
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
		if err := database.playerGetOrCreate(ctx, team.TeamLeader, &record); err != nil {
			return team, err
		}

		if err := rglTeamInsert(ctx, database, team); err != nil {
			return team, err
		}
	}

	return team, nil
}

func rglTeamGet(ctx context.Context, database *pgStore, teamID int) (RGLTeam, error) {
	var team RGLTeam
	query, args, errQuery := sb.
		Select("team_id", "season_id", "division_id", "division_name", "team_leader", "tag", "team_name", "final_rank",
			"created_at", "updated_at").
		From("rgl_team").
		Where(sq.Eq{"team_id": teamID}).
		ToSql()
	if errQuery != nil {
		return team, dbErr(errQuery, "Failed to build query")
	}

	if err := database.pool.QueryRow(ctx, query, args...).
		Scan(&team.TeamID, &team.SeasonID, &team.DivisionID, &team.DivisionName, &team.TeamLeader, &team.Tag, &team.TeamName, &team.FinalRank,
			&team.CreatedAt, &team.UpdatedAt); err != nil {
		return team, dbErr(err, "Failed to query rgl team")
	}

	return team, nil
}

func rglTeamInsert(ctx context.Context, database *pgStore, team RGLTeam) error {
	query, args, errQuery := sb.
		Insert("rgl_team").
		SetMap(map[string]interface{}{
			"team_id":       team.TeamID,
			"season_id":     team.SeasonID,
			"division_id":   team.DivisionID,
			"division_name": team.DivisionName,
			"team_leader":   team.TeamLeader,
			"tag":           team.Tag,
			"team_name":     team.TeamName,
			"final_rank":    team.FinalRank,
			"created_at":    team.CreatedAt,
			"updated_at":    team.UpdatedAt,
		}).ToSql()
	if errQuery != nil {
		return dbErr(errQuery, "Failed to insert rgl team")
	}

	if _, err := database.pool.Exec(ctx, query, args...); err != nil {
		return dbErr(err, "Failed to exec rgl team insert")
	}

	return nil
}

func getRGLSeason(ctx context.Context, database *pgStore, seasonID int) (RGLSeason, error) {
	season, errSeason := rglSeasonGet(ctx, database, seasonID)
	if errSeason != nil && !errors.Is(errSeason, errDatabaseNoResults) {
		return season, errSeason
	}

	if season.SeasonID == 0 {
		fetchedSeason, errFetch := rgl.Season(ctx, NewHTTPClient(), int64(seasonID))
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

		if err := rglSeasonInsert(ctx, database, season); err != nil {
			return season, err
		}
	}

	return season, nil
}

func rglSeasonGet(ctx context.Context, database *pgStore, seasonID int) (RGLSeason, error) {
	var season RGLSeason
	query, args, errQuery := sb.
		Select("season_id", "maps", "season_name", "format_name", "region_name", "participating_teams", "created_on").
		From("rgl_season").
		Where(sq.Eq{"season_id": seasonID}).
		ToSql()
	if errQuery != nil {
		return season, dbErr(errQuery, "Failed to build query")
	}

	if err := database.pool.QueryRow(ctx, query, args...).
		Scan(&season.SeasonID, &season.Maps, &season.Name, &season.FormatName, &season.RegionName, &season.ParticipatingTeams, &season.CreatedOn); err != nil {
		return season, dbErr(err, "Failed to query rgl season")
	}

	return season, nil
}

func rglSeasonInsert(ctx context.Context, database *pgStore, season RGLSeason) error {
	query, args, errQuery := sb.
		Insert("rgl_season").
		SetMap(map[string]interface{}{
			"season_id":           season.SeasonID,
			"maps":                season.Maps,
			"season_name":         season.Name,
			"format_name":         season.FormatName,
			"region_name":         season.RegionName,
			"participating_teams": season.ParticipatingTeams,
			"created_on":          season.CreatedOn,
		}).ToSql()
	if errQuery != nil {
		return dbErr(errQuery, "Failed to insert rgl season")
	}

	if _, err := database.pool.Exec(ctx, query, args...); err != nil {
		return dbErr(err, "Failed to exec rgl season insert")
	}

	return nil
}
