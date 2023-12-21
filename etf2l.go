package main

import (
	"context"
	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/etf2l"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"time"
)

type TimeStamped struct {
	CreatedOn time.Time `json:"created_on"`
	UpdatedOn time.Time `json:"updated_on"`
}

func newTimeStamped() TimeStamped {
	t0 := time.Now()
	return TimeStamped{UpdatedOn: t0, CreatedOn: t0}
}

type ETF2LPlayer struct {
	SteamID steamid.SID64 `json:"steam_id"`
	ID      int           `json:"id"`
	Name    string        `json:"name"`
	Country string        `json:"country"`
	TimeStamped
}

type ETF2LBan struct {
	BanID     int           `json:"ban_id"`
	SteamID   steamid.SID64 `json:"steam_id"`
	StartDate time.Time     `json:"start_date"`
	EndDate   time.Time     `json:"end_date"`
	Reason    string        `json:"reason"`
	TimeStamped
}

type ETF2LTeam struct {
	TeamID   int    `json:"id"`
	Name     string `json:"name"`
	Title    string `json:"title"`
	Country  string `json:"country"`
	TeamType string `json:"type"`
	TimeStamped
}

type ETF2LCompetition struct {
	CompetitionID int    `json:"competition_id"`
	TeamID        int    `json:"team_id"`
	Category      string `json:"category"`
	Competition   string `json:"competition"`
	DivisionName  string `json:"division_name"`
	DivisionTier  int    `json:"division_tier"`
	TimeStamped
}

func etf2lSavePlayer(ctx context.Context, db *pgStore, player *etf2l.Player) error {
	now := time.Now()

	return db.ExecInsertBuilder(ctx, sb.
		Insert("etf2l_player").
		SetMap(map[string]interface{}{
			"steam_id":   player.Steam.ID64.Int64(),
			"id":         player.ID,
			"name":       player.Name,
			"country":    player.Country,
			"created_on": now,
			"updated_on": now,
		}).Suffix("ON CONFLICT DO NOTHING"))
}

func etf2lPlayerBySteamID(ctx context.Context, db *pgStore, steamID steamid.SID64, player *ETF2LPlayer) error {
	builder := sb.
		Select("id", "name", "country", "created_on", "updated_on").
		From("etf2l_player").
		Where(sq.Eq{"steam_id": steamID.Int64()})

	row, errRow := db.QueryRowBuilder(ctx, builder)
	if errRow != nil {
		return errRow
	}

	if errScan := row.Scan(&player.ID, &player.Name, &player.Country, &player.CreatedOn, &player.UpdatedOn); errScan != nil {
		return Err(errScan)
	}

	player.SteamID = steamID

	return nil
}

func etf2lPlayerByID(ctx context.Context, db *pgStore, playerID int, player *ETF2LPlayer) error {
	builder := sb.
		Select("steam_id", "id", "name", "country", "created_on", "updated_on").
		From("etf2l_player").
		Where(sq.Eq{"id": playerID})

	row, errRow := db.QueryRowBuilder(ctx, builder)
	if errRow != nil {
		return Err(errRow)
	}

	var steamID int64

	if errScan := row.Scan(&steamID, &player.ID, &player.Name, &player.Country, &player.CreatedOn, &player.UpdatedOn); errScan != nil {
		return Err(errScan)
	}

	player.SteamID = steamid.New(steamID)

	return nil
}

func etf2lSaveBan(ctx context.Context, db *pgStore, steamID steamid.SID64, ban etf2l.BanReason) error {
	now := time.Now()

	return db.ExecInsertBuilder(ctx, sb.
		Insert("etf2l_bans").
		SetMap(map[string]interface{}{
			"steam_id":   steamID.Int64(),
			"start_date": time.Unix(int64(ban.Start), 0),
			"end_date":   time.Unix(int64(ban.End), 0),
			"reason":     ban.Reason,
			"created_on": now,
			"updated_on": now,
		}).
		Suffix("ON CONFLICT DO NOTHING"))
}

func etf2lBans(ctx context.Context, db *pgStore, steamID steamid.SID64) ([]ETF2LBan, error) {
	rows, err := db.QueryBuilder(ctx, sb.
		Select("ban_id", "steam_id", "start_date", "end_date", "reason", "created_on", "updated_on").
		From("etf2l_bans").
		Where(sq.Eq{"steam_id": steamID.Int64()}))
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var bans []ETF2LBan

	for rows.Next() {
		var ban ETF2LBan
		if errScan := rows.Scan(&ban.BanID, &ban.SteamID, &ban.StartDate, &ban.EndDate, &ban.Reason, &ban.CreatedOn, &ban.UpdatedOn); errScan != nil {
			return nil, errScan
		}

		bans = append(bans, ban)
	}

	return bans, nil
}

func etf2lTeamPlayerSave(ctx context.Context, db *pgStore, steamID steamid.SID64, teamID int) error {
	now := time.Now()
	query := sb.
		Insert("etf2l_team_player").
		SetMap(map[string]interface{}{
			"team_id":    teamID,
			"steam_id":   steamID.Int64(),
			"created_on": now,
			"updated_on": now,
		}).Suffix("ON CONFLICT DO NOTHING")

	return db.ExecInsertBuilder(ctx, query)
}

func etf2lTeamSave(ctx context.Context, db *pgStore, team etf2l.Team) error {
	now := time.Now()
	query := sb.
		Insert("etf2l_team").
		SetMap(map[string]interface{}{
			"team_id":    team.ID,
			"name":       team.Name,
			"country":    team.Country,
			"created_on": now,
			"updated_on": now,
		}).Suffix("ON CONFLICT DO NOTHING")

	return db.ExecInsertBuilder(ctx, query)
}

func etf2lTeams(ctx context.Context, db *pgStore, teamIDS []int) ([]ETF2LTeam, error) {
	rows, err := db.QueryBuilder(ctx, sb.
		Select("team_id", "name", "country", "created_on", "updated_on").
		From("etf2l_team").
		Where(sq.Eq{"id": teamIDS}))
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var teams []ETF2LTeam

	for rows.Next() {
		var team ETF2LTeam
		if errScan := rows.Scan(&team.TeamID, &team.Name, &team.Country, &team.CreatedOn, &team.UpdatedOn); errScan != nil {
			return nil, errScan
		}

		teams = append(teams, team)
	}

	return teams, nil
}

func etf2lTeam(ctx context.Context, db *pgStore, teamID int) (*ETF2LTeam, error) {
	row, err := db.QueryRowBuilder(ctx, sb.
		Select("team_id", "name", "country", "created_on", "updated_on").
		From("etf2l_team").
		Where(sq.Eq{"team_id": teamID}))
	if err != nil {
		return nil, err
	}

	var team ETF2LTeam
	if errScan := row.Scan(&team.TeamID, &team.Name, &team.Country, &team.CreatedOn, &team.UpdatedOn); errScan != nil {
		return nil, Err(errScan)
	}

	return &team, nil
}

func etf2lPlayerTeams(ctx context.Context, db *pgStore, steamID steamid.SID64) ([]ETF2LTeam, error) {
	rows, err := db.QueryBuilder(ctx, sb.
		Select("t.team_id", "t.name", "t.country", "t.created_on", "t.updated_on").
		From("etf2l_team t").
		LeftJoin("etf2l_team_player p USING(team_id)").
		Where(sq.Eq{"p.steam_id": steamID.Int64()}))
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var teams []ETF2LTeam

	for rows.Next() {
		var team ETF2LTeam
		if errScan := rows.Scan(&team.TeamID, &team.Name, &team.Country, &team.CreatedOn, &team.UpdatedOn); errScan != nil {
			return nil, errScan
		}

		teams = append(teams, team)
	}

	return teams, nil
}

func etf2lSaveTeam(ctx context.Context, db *pgStore, team *etf2l.Team) error {
	now := time.Now()
	builder := sb.
		Insert("etf2l_team").
		SetMap(map[string]interface{}{
			"team_id": team.ID,
			"name":    team.Name,
			//"type":       team.Type,
			"tag":        team.Tag,
			"country":    team.Country,
			"created_on": now,
			"updated_on": now,
		}).Suffix("ON CONFLICT DO NOTHING")
	return db.ExecInsertBuilder(ctx, builder)
}

func etf2lSaveCompetition(ctx context.Context, db *pgStore, competitionID int, teamID int, team etf2l.TeamCompetition) error {
	now := time.Now()

	return db.ExecInsertBuilder(ctx, sb.
		Insert("etf2l_competition").
		SetMap(map[string]interface{}{
			"competition_id": competitionID,
			"team_id":        teamID,
			"category":       team.Category,
			"competition":    team.Competition,
			"division_name":  team.Division.Name,
			"division_tier":  team.Division.Tier,
			"created_on":     now,
			"updated_on":     now,
		}).
		Suffix("ON CONFLICT DO NOTHING"))
}

func etf2lCompetitions(ctx context.Context, db *pgStore, teamID int) ([]ETF2LCompetition, error) {
	rows, err := db.QueryBuilder(ctx, sb.
		Select("competition_id", "team_id", "category", "competition", "division_name", "division_tier", "created_on", "updated_on").
		From("etf2l_competition").
		Where(sq.Eq{"team_id": teamID}))
	if err != nil {
		return nil, Err(err)
	}

	defer rows.Close()

	var comps []ETF2LCompetition

	for rows.Next() {
		var comp ETF2LCompetition
		if errScan := rows.Scan(&comp.CompetitionID, &comp.TeamID, &comp.Competition, &comp.Competition,
			&comp.DivisionName, &comp.DivisionTier, &comp.CreatedOn, &comp.UpdatedOn); errScan != nil {
			return nil, errScan
		}

		comps = append(comps, comp)
	}

	return comps, nil
}

func etf2lCompetition(ctx context.Context, db *pgStore, competitionID int, comp *ETF2LCompetition) error {
	rows, err := db.QueryRowBuilder(ctx, sb.
		Select("competition_id", "team_id", "category", "competition", "division_name", "division_tier", "created_on", "updated_on").
		From("etf2l_competition").
		Where(sq.Eq{"competition_id": competitionID}))
	if err != nil {
		return Err(err)
	}

	if errScan := rows.Scan(&comp.CompetitionID, &comp.TeamID, &comp.Competition, &comp.Competition,
		&comp.DivisionName, &comp.DivisionTier, &comp.CreatedOn, &comp.UpdatedOn); errScan != nil {
		return errScan
	}

	return nil
}

//func etf2lSaveCompetitionTable(ctx context.Context, db *pgStore, competitionID int, teamID int, team etf2l.CompetitionTable) error {
//	now := time.Now()
//
//	return db.ExecInsertBuilder(ctx, sb.
//		Insert("etf2l_competition_table").
//		SetMap(map[string]interface{}{
//			"competition_id": competitionID,
//			"team_id":        teamID,
//			"category":       team.Category,
//			"competition":    team.Competition,
//			"division_name":  team.DivisionName,
//			"division_tier":  team.DivisionID,
//			"created_on":     now,
//			"updated_on":     now,
//		}).
//		Suffix("ON CONFLICT DO NOTHING"))
//}
