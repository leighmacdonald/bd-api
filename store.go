package main

import (
	"context"
	"database/sql"
	"embed"
	sq "github.com/Masterminds/squirrel"
	"github.com/golang-migrate/migrate/v4"
	pgxMigrate "github.com/golang-migrate/migrate/v4/database/pgx"
	"github.com/golang-migrate/migrate/v4/source/httpfs"
	"github.com/jackc/pgx/v4/pgxpool"
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

	dbConn, errConnectConfig := pgxpool.ConnectConfig(ctx, cfg)
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

type playerRecord struct {
	SteamID                  string    `json:"steam_id"`
	CommunityVisibilityState int       `json:"community_visibility_state"`
	ProfileState             int       `json:"profile_state"`
	PersonaName              string    `json:"persona_name"`
	Vanity                   string    `json:"vanity"`
	AvatarHash               string    `json:"avatar_hash"`
	PersonaState             int       `json:"persona_state"`
	RealName                 string    `json:"real_name"`
	TimeCreated              int       `json:"time_created"`
	LocCountryCode           string    `json:"loc_country_code"`
	LocStateCode             string    `json:"loc_state_code"`
	LocCityID                int       `json:"loc_city_id"`
	CommunityBanned          bool      `json:"community_banned"`
	VacBanned                bool      `json:"vac_banned"`
	GameBans                 int       `json:"game_bans"`
	EconomyBanned            int       `json:"economy_banned"`
	LogsTfCount              int       `json:"logs_tf_count"`
	UGCUpdatedOn             time.Time `json:"ugc_updated_on"`
	RGLUpdatedOn             time.Time `json:"rgl_updated_on"`
	ETF2LUpdatedOn           time.Time `json:"etf2l_updated_on"`
	LogsTFUpdatedOn          time.Time `json:"logs_tf_updated_on"`
	timeStamped
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
	if s.SiteID > 0 {
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
