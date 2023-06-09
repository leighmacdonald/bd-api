package main

import (
	"context"
	"database/sql"
	"embed"
	sq "github.com/Masterminds/squirrel"
	"github.com/golang-migrate/migrate/v4"
	pgxMigrate "github.com/golang-migrate/migrate/v4/database/pgx"
	"github.com/golang-migrate/migrate/v4/source/httpfs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"net/http"
	"time"
)

var (
	// ErrNoResult is returned on successful queries which return no rows
	//errNoResult = errors.New("No results found")
	// ErrDuplicate is returned when a duplicate row result is attempted to be inserted
	//errDuplicate = errors.New("Duplicate entity")
	// Use $ for pg based queries
	sb = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	//go:embed migrations
	migrations embed.FS
)

func newStore(ctx context.Context, dsn string) (*pgStore, error) {
	log := logger.Named("db")
	cfg, errConfig := pgxpool.ParseConfig(dsn)
	if errConfig != nil {
		return nil, errors.Errorf("Unable to parse config: %v", errConfig)
	}
	database := pgStore{
		logger: log,
		dsn:    dsn,
	}
	if errMigrate := database.migrate(); errMigrate != nil {
		if errMigrate.Error() == "no change" {
			database.logger.Info("Migration at latest version")
		} else {
			return nil, errors.Errorf("Could not migrate schema: %v", errMigrate)
		}
	} else {
		database.logger.Info("Migration completed successfully")
	}

	dbConn, errConnectConfig := pgxpool.NewWithConfig(ctx, cfg)
	if errConnectConfig != nil {
		return nil, errors.Wrap(errConnectConfig, "Failed to connect to database")
	}
	database.pool = dbConn
	return &database, nil
}

type pgStore struct {
	dsn    string
	logger *zap.Logger
	pool   *pgxpool.Pool
}

// Migrate database schema
func (db *pgStore) migrate() error {
	instance, errOpen := sql.Open("pgx", db.dsn)
	if errOpen != nil {
		return errors.Wrapf(errOpen, "Failed to open database for migration")
	}
	if errPing := instance.Ping(); errPing != nil {
		return errors.Wrapf(errPing, "Cannot migrate, failed to connect to target server")
	}
	driver, errMigrate := pgxMigrate.WithInstance(instance, &pgxMigrate.Config{
		MigrationsTable:       "_migration",
		SchemaName:            "public",
		StatementTimeout:      60 * time.Second,
		MultiStatementEnabled: true,
	})
	if errMigrate != nil {
		return errors.Wrapf(errMigrate, "failed to create migration driver")
	}
	defer logCloser(driver)
	source, errHTTPFs := httpfs.New(http.FS(migrations), "migrations")
	if errHTTPFs != nil {
		return errHTTPFs
	}
	migrator, errMigrateInstance := migrate.NewWithInstance("iofs", source, "pgx", driver)
	if errMigrateInstance != nil {
		return errors.Wrapf(errMigrateInstance, "Failed to migrator up")
	}
	return migrator.Up()
}

type timeStamped struct {
	UpdatedOn time.Time `json:"steam_updated_on"`
	CreatedOn time.Time `json:"created_on"`
}

type playerNameRecord struct {
	NameID      int64         `json:"name_id"`
	SteamID     steamid.SID64 `json:"steam_id"`
	PersonaName string        `json:"persona_name"`
	CreatedOn   time.Time     `json:"created_on"`
}

type playerAvatarRecord struct {
	NameID     int64         `json:"name_id"`
	SteamID    steamid.SID64 `json:"steam_id"`
	AvatarHash string        `json:"avatar_hash"`
	CreatedOn  time.Time     `json:"created_on"`
}

type playerVanityRecord struct {
	NameID    int64         `json:"name_id"`
	SteamID   steamid.SID64 `json:"steam_id"`
	Vanity    string        `json:"vanity"`
	CreatedOn time.Time     `json:"created_on"`
}

type playerRecord struct {
	SteamID                  steamid.SID64 `json:"steam_id"`
	CommunityVisibilityState int           `json:"community_visibility_state"`
	ProfileState             int           `json:"profile_state"`
	PersonaName              string        `json:"persona_name"`
	Vanity                   string        `json:"vanity"`
	AvatarHash               string        `json:"avatar_hash"`
	PersonaState             int           `json:"persona_state"`
	RealName                 string        `json:"real_name"`
	TimeCreated              int           `json:"time_created"`
	LocCountryCode           string        `json:"loc_country_code"`
	LocStateCode             string        `json:"loc_state_code"`
	LocCityID                int           `json:"loc_city_id"`
	CommunityBanned          bool          `json:"community_banned"`
	VacBanned                bool          `json:"vac_banned"`
	GameBans                 int           `json:"game_bans"`
	EconomyBanned            int           `json:"economy_banned"`
	LogsTFCount              int           `json:"logs_tf_count"`
	UGCUpdatedOn             time.Time     `json:"ugc_updated_on"`
	RGLUpdatedOn             time.Time     `json:"rgl_updated_on"`
	ETF2LUpdatedOn           time.Time     `json:"etf2l_updated_on"`
	LogsTFUpdatedOn          time.Time     `json:"logs_tf_updated_on"`
	isNewRecord              bool
	timeStamped
}

const defaultAvatar = "fef49e7fa7e1997310d705b2a6158ff8dc1cdfeb"

func newPlayerRecord(sid64 steamid.SID64) playerRecord {
	t0 := time.Now()
	return playerRecord{
		SteamID:     sid64,
		isNewRecord: true,
		AvatarHash:  defaultAvatar,
		timeStamped: timeStamped{
			UpdatedOn: t0,
			CreatedOn: t0,
		}}
}

func playerNameSave(ctx context.Context, tx pgx.Tx, r *playerRecord) error {
	query, args, errSQL := sb.Insert("player_names").Columns("steam_id", "persona_name").Values(r.SteamID, r.PersonaName).ToSql()
	if errSQL != nil {
		return errSQL
	}
	if _, errName := tx.Exec(ctx, query, args...); errName != nil {
		return errName
	}
	return nil
}

func playerAvatarSave(ctx context.Context, tx pgx.Tx, r *playerRecord) error {
	query, args, errSQL := sb.Insert("player_avatars").Columns("steam_id", "avatar_hash").Values(r.SteamID, r.AvatarHash).ToSql()
	if errSQL != nil {
		return errSQL
	}
	if _, errName := tx.Exec(ctx, query, args...); errName != nil {
		return errName
	}
	return nil
}

func playerVanitySave(ctx context.Context, tx pgx.Tx, r *playerRecord) error {
	query, args, errSQL := sb.Insert("player_vanity").Columns("steam_id", "vanity").Values(r.SteamID, r.Vanity).ToSql()
	if errSQL != nil {
		return errSQL
	}
	if _, errName := tx.Exec(ctx, query, args...); errName != nil {
		return errName
	}
	return nil
}

func (db *pgStore) playerRecordSave(ctx context.Context, r *playerRecord) error {
	tx, errTx := db.pool.BeginTx(ctx, pgx.TxOptions{})
	if errTx != nil {
		return errTx
	}
	defer func() {
		if errRollback := tx.Rollback(ctx); errRollback != nil {
			logger.Error("Failed to rollback player tx", zap.Error(errRollback))
		}
	}()
	if r.isNewRecord {
		query, args, errSQL := sb.
			Insert("player").
			Columns("steam_id", "community_visibility_state", "profile_state", "persona_name", "vanity",
				"avatar_hash", "persona_state", "real_name", "time_created", "loc_country_code", "loc_state_code", "loc_city_id",
				"community_banned", "vac_banned", "game_bans", "economy_banned", "logs_tf_count", "ugc_updated_on", "rgl_updated_on",
				"etf2l_updated_on", "logs_tf_updated_on", "updated_on", "created_on").
			Values(r.SteamID, r.CommunityVisibilityState, r.ProfileState, r.PersonaName, r.Vanity,
				r.AvatarHash, r.PersonaState, r.RealName, r.TimeCreated, r.LocCountryCode, r.LocStateCode, r.LocCityID,
				r.CommunityBanned, r.VacBanned, r.GameBans, r.EconomyBanned, r.LogsTFCount, r.UGCUpdatedOn, r.RGLUpdatedOn,
				r.ETF2LUpdatedOn, r.LogsTFUpdatedOn, r.UpdatedOn, r.CreatedOn).
			ToSql()
		if errSQL != nil {
			return errSQL
		}
		if _, errExec := tx.Exec(ctx, query, args...); errExec != nil {
			return errExec
		}
		r.isNewRecord = false
		if errName := playerNameSave(ctx, tx, r); errName != nil {
			return errName
		}
		if errVanity := playerVanitySave(ctx, tx, r); errVanity != nil {
			return errVanity
		}
		if errAvatar := playerAvatarSave(ctx, tx, r); errAvatar != nil {
			return errAvatar
		}
	} else {
		query, args, errSQL := sb.
			Update("player").
			Set("steam_id", r.SteamID).
			Set("community_visibility_state", r.CommunityVisibilityState).
			Set("profile_state", r.ProfileState).
			Set("persona_name", r.PersonaName).
			Set("vanity", r.Vanity).
			Set("avatar_hash", r.AvatarHash).
			Set("persona_state", r.PersonaState).
			Set("real_name", r.RealName).
			Set("time_created", r.TimeCreated).
			Set("loc_country_code", r.LocCountryCode).
			Set("loc_state_code", r.LocStateCode).
			Set("loc_city_id", r.LocCityID).
			Set("community_banned", r.CommunityBanned).
			Set("vac_banned", r.VacBanned).
			Set("game_bans", r.GameBans).
			Set("economy_banned", r.EconomyBanned).
			Set("logs_tf_count", r.LogsTFCount).
			Set("ugc_updated_on", r.UGCUpdatedOn).
			Set("rgl_updated_on", r.RGLUpdatedOn).
			Set("etf2l_updated_on", r.ETF2LUpdatedOn).
			Set("logs_tf_updated_on", r.LogsTFUpdatedOn).
			Set("updated_on", r.UpdatedOn).
			Where(sq.Eq{"steam_id": r.SteamID}).
			ToSql()
		if errSQL != nil {
			return errSQL
		}
		if _, errExec := tx.Exec(ctx, query, args...); errExec != nil {
			return errExec
		}
	}
	if errCommit := tx.Commit(ctx); errCommit != nil {
		logger.Error("Failed to commit player tx", zap.Error(errCommit))
	}
	return nil
}

//type leagueRecord struct {
//	LeagueID   int       `json:"league_id"`
//	LeagueName string    `json:"league_name"`
//	UpdatedOn  time.Time `json:"Updated_on"`
//	CreatedOn  time.Time `json:"created_on"`
//}
//
//type teamRecord struct {
//}

type sbSite struct {
	SiteID int    `json:"site_id"`
	Name   string `json:"name"`
	timeStamped
}

func newSBSite(name string) sbSite {
	t0 := time.Now()
	return sbSite{
		SiteID: 0,
		Name:   name,
		timeStamped: timeStamped{
			UpdatedOn: t0,
			CreatedOn: t0,
		},
	}
}

type sbBanRecord struct {
	BanID     int           `json:"ban_id"`
	SiteID    string        `json:"site_id"`
	SteamID   steamid.SID64 `json:"steam_id"`
	Reason    string        `json:"reason"`
	Duration  time.Duration `json:"duration"`
	Permanent bool          `json:"permanent"`
	timeStamped
}

func (db *pgStore) sbSiteSave(ctx context.Context, s *sbSite) error {
	s.UpdatedOn = time.Now()
	if s.SiteID <= 0 {
		s.CreatedOn = time.Now()
		query, args, errSQL := sb.
			Insert("sb_site").
			Columns("name", "updated_on", "created_on").
			Values(s.Name, s.UpdatedOn, s.CreatedOn).
			Suffix("RETURNING sb_site_id").
			ToSql()
		if errSQL != nil {
			return errSQL
		}
		if errQuery := db.pool.QueryRow(ctx, query, args...).Scan(&s.SiteID); errQuery != nil {
			return errQuery
		}
	} else {
		query, args, errSQL := sb.
			Update("sb_site").
			Set("name", s.Name).
			Set("updated_on", s.UpdatedOn).
			ToSql()
		if errSQL != nil {
			return errSQL
		}
		if _, errQuery := db.pool.Exec(ctx, query, args...); errQuery != nil {
			return errQuery
		}
	}
	return nil
}

func (db *pgStore) sbSiteGet(ctx context.Context, siteId int, site *sbSite) error {
	query, args, errSQL := sb.
		Select("sb_site_id", "name", "updated_on", "created_on").
		From("sb_site").
		Where(sq.Eq{"sb_site_id": siteId}).
		ToSql()
	if errSQL != nil {
		return errSQL
	}
	if errQuery := db.pool.QueryRow(ctx, query, args...).Scan(&site.SiteID, &site.Name, &site.UpdatedOn, &site.CreatedOn); errQuery != nil {
		return errQuery
	}
	return nil
}

func (db *pgStore) sbSiteDelete(ctx context.Context, siteId int) error {
	query, args, errSQL := sb.
		Delete("sb_site").
		Where(sq.Eq{"sb_site_id": siteId}).
		ToSql()
	if errSQL != nil {
		return errSQL
	}
	if _, errQuery := db.pool.Exec(ctx, query, args...); errQuery != nil {
		return errQuery
	}
	return nil
}

func (db *pgStore) sbBanSave(ctx context.Context, s *sbBanRecord) error {
	s.UpdatedOn = time.Now()
	if s.BanID > 0 {
		s.CreatedOn = time.Now()
		query, args, errSQL := sb.
			Insert("sb_ban").
			Columns("sb_site_id", "steam_id", "reason", "created_on", "duration", "permanent").
			Values(s.SiteID, s.SteamID, s.Reason, s.CreatedOn, s.Duration, s.Permanent).
			Suffix("RETURNING sb_ban_id").
			ToSql()
		if errSQL != nil {
			return errSQL
		}
		if errQuery := db.pool.QueryRow(ctx, query, args...).Scan(&s.BanID); errQuery != nil {
			return errQuery
		}
	} else {
		query, args, errSQL := sb.
			Update("sb_site").
			Set("sb_site_id", s.SiteID).
			Set("steam_id", s.SteamID).
			Set("reason", s.Reason).
			Set("created_on", s.CreatedOn).
			Set("duration", s.Duration).
			Set("permanent", s.Permanent).
			ToSql()
		if errSQL != nil {
			return errSQL
		}
		if _, errQuery := db.pool.Exec(ctx, query, args...); errQuery != nil {
			return errQuery
		}
	}
	return nil
}
