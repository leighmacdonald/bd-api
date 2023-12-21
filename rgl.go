package main

import (
	"context"
	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/rgl"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"time"
)

func parseRGLDivision(div string) Division {
	switch div {
	case "RGL-Invite":
		fallthrough
	case "Invite":
		return RGLRankInvite
	case "RGL-Div-1":
		return RGLRankDiv1
	case "RGL-Div-2":
		return RGLRankDiv2
	case "RGL-Main":
		fallthrough
	case "Main":
		return RGLRankMain
	case "RGL-Advanced":
		fallthrough
	case "Advanced-1":
		fallthrough
	case "Advanced":
		return RGLRankAdvanced
	case "RGL-Intermediate":
		fallthrough
	case "Intermediate":
		return RGLRankIntermediate
	case "RGL-Challenger":
		return RGLRankIntermediate
	case "Open":
		return RGLRankOpen
	case "Amateur":
		return RGLRankAmateur
	case "Fresh Meat":
		return RGLRankFreshMeat
	default:
		return RGLRankNone
	}
}

type RGLTeam struct {
	TeamID       int       `json:"team_id"`
	Name         string    `json:"name"`
	Tag          string    `json:"tag"`
	DivisionName string    `json:"division_name"`
	DivisionID   int       `json:"division_id"`
	FinalRank    int       `json:"final_rank"`
	SeasonID     int       `json:"season_id"`
	CreatedOn    time.Time `json:"created_on"`
	UpdatedOn    time.Time `json:"updated_on"`
}

func rglTeam(ctx context.Context, db *pgStore, teamID int, team *RGLTeam) error {
	row, errRow := db.QueryRowBuilder(ctx, sb.
		Select("team_id", "name", "tag", "division_name", "division_id",
			"final_rank", "season_id", "created_on", "updated_on").
		From("rgl_team").Where(sq.Eq{"team_id": teamID}))
	if errRow != nil {
		return errRow
	}

	return Err(row.Scan(&team.TeamID, &team.Name, &team.Tag, &team.DivisionName, &team.DivisionID, &team.FinalRank,
		&team.SeasonID, &team.CreatedOn, &team.UpdatedOn))
}

func rglTeamSave(ctx context.Context, db *pgStore, overview *rgl.TeamOverview) error {
	return Err(db.ExecInsertBuilder(ctx, sb.Insert("rgl_team").SetMap(map[string]interface{}{
		"team_id":       overview.TeamID,
		"name":          overview.Name,
		"tag":           overview.Tag,
		"division_name": overview.DivisionName,
		"division_id":   overview.DivisionID,
		"final_rank":    overview.FinalRank,
		"season_id":     overview.SeasonID,
		"created_on":    overview.CreatedAt,
		"updated_on":    overview.UpdatedAt,
	})))
}

type RGLPlayer struct {
	SteamID       steamid.SID64 `json:"steam_id"`
	Avatar        string        `json:"avatar"`
	Name          string        `json:"name"`
	UpdatedAt     time.Time     `json:"updated_at"`
	IsVerified    bool          `json:"is_verified"`
	IsBanned      bool          `json:"is_banned"`
	IsOnProbation bool          `json:"is_on_probation"`
	CreatedOn     time.Time     `json:"created_on"`
	UpdatedOn     time.Time     `json:"updated_on"`
}

func rglPlayer(ctx context.Context, db *pgStore, steamID steamid.SID64, player *RGLPlayer) error {
	row, errRow := db.QueryRowBuilder(ctx, sb.
		Select("avatar", "name", "updated_at",
			"is_verified", "is_banned", "is_on_probation", "created_on", "updated_on").
		From("rgl_player").Where(sq.Eq{"steam_id": steamID.Int64()}))
	if errRow != nil {
		return errRow
	}
	player.SteamID = steamID

	return Err(row.Scan(&player.Avatar, &player.Name, &player.UpdatedAt, &player.IsVerified,
		&player.IsBanned, &player.IsOnProbation, &player.CreatedOn, &player.UpdatedOn))
}

func rglPlayerSave(ctx context.Context, db *pgStore, player *rgl.Player) error {
	now := time.Now()

	return db.ExecInsertBuilder(ctx, sb.
		Insert("rgl_player").
		SetMap(map[string]interface{}{
			"steam_id":        player.SteamID.Int64(),
			"avatar":          player.Avatar,
			"name":            player.Name,
			"updated_at":      player.UpdatedAt,
			"is_verified":     player.Status.IsVerified,
			"is_banned":       player.Status.IsBanned,
			"is_on_probation": player.Status.IsOnProbation,
			"created_on":      now,
			"updated_on":      now,
		}).Suffix("ON CONFLICT DO NOTHING"))
}

func rglTeamPlayerSave(ctx context.Context, db *pgStore, teamID int, player rgl.TeamPlayer) error {
	now := time.Now()

	return db.ExecInsertBuilder(ctx, sb.
		Insert("rgl_team_player").
		SetMap(map[string]interface{}{
			"team_id":    teamID,
			"steam_id":   player.SteamID.Int64(),
			"name":       player.Name,
			"is_leader":  player.IsLeader,
			"joined_at":  player.JoinedAt,
			"left_at":    player.LeftAt,
			"created_on": now,
			"updated_on": now,
		}).
		Suffix("ON CONFLICT DO NOTHING"))
}

type RGLBan struct {
	SteamID   steamid.SID64 `json:"steam_id"`
	Name      string        `json:"name"`
	Reason    string        `json:"reason"`
	CreatedAt time.Time     `json:"created_at"`
	ExpiresAt time.Time     `json:"expires_at"`
	CreatedOn time.Time     `json:"created_on"`
	UpdatedOn time.Time     `json:"updated_on"`
}

func rglBansBySteamID(ctx context.Context, db *pgStore, steamID steamid.SID64) ([]RGLBan, error) {
	rows, errRows := db.QueryBuilder(ctx, sb.
		Select("name", "reason", "created_at", "expired_at", "created_on", "updated_on").
		From("rgl_ban").
		Where(sq.Eq{"steam_id": steamID.Int64()}))
	if errRows != nil {
		return nil, errRows
	}

	defer rows.Close()

	var bans []RGLBan

	for rows.Next() {
		ban := RGLBan{SteamID: steamID}
		if errScan := rows.Scan(&ban.Name, &ban.Reason, &ban.CreatedOn, &ban.ExpiresAt, &ban.CreatedOn, &ban.UpdatedOn); errScan != nil {
			return nil, errScan
		}

		bans = append(bans, ban)
	}

	return bans, nil
}

func rglBanSave(ctx context.Context, db *pgStore, ban rgl.Ban) error {
	now := time.Now()

	return db.ExecInsertBuilder(ctx, sb.
		Insert("rgl_ban").
		SetMap(map[string]interface{}{
			"steam_id":   steamid.New(ban.SteamID).Int64(),
			"name":       ban.Alias,
			"reason":     ban.Reason,
			"created_at": ban.CreatedAt,
			"expires_at": ban.ExpiresAt,
			"created_on": now,
			"updated_on": now,
		}).
		Suffix("ON CONFLICT DO NOTHING"))
}
