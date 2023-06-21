package main

import (
	"context"
	"database/sql"
	"embed"
	"net/http"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/jackc/pgx/v5"
	"github.com/leighmacdonald/steamweb/v2"

	sq "github.com/Masterminds/squirrel"
	pgxMigrate "github.com/golang-migrate/migrate/v4/database/pgx"
	"github.com/golang-migrate/migrate/v4/source/httpfs"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var (
	// ErrNoResult is returned on successful queries which return no rows.

	// ErrDuplicate is returned when a duplicate row result is attempted to be inserted
	// errDuplicate = errors.New("Duplicate entity")
	// Use $ for pg based queries.
	sb = sq.StatementBuilder.PlaceholderFormat(sq.Dollar) //nolint:gochecknoglobals
	//go:embed migrations
	migrations embed.FS

	errDuplicate = errors.New("Duplicate entity")
	errNoRows    = errors.New("No rows")
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
		pool:   nil,
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

// Migrate database schema.
func (db *pgStore) migrate() error {
	const stmtTimeout = 60 * time.Second

	instance, errOpen := sql.Open("pgx", db.dsn)
	if errOpen != nil {
		return errors.Wrapf(errOpen, "Failed to open database for migration")
	}

	if errPing := instance.Ping(); errPing != nil {
		return errors.Wrapf(errPing, "Cannot migrate, failed to connect to target server")
	}

	driver, errMigrate := pgxMigrate.WithInstance(instance, &pgxMigrate.Config{ //nolint:exhaustruct
		MigrationsTable:       "_migration",
		SchemaName:            "public",
		StatementTimeout:      stmtTimeout,
		MultiStatementEnabled: true,
	})
	if errMigrate != nil {
		return errors.Wrapf(errMigrate, "failed to create migration driver")
	}

	defer logCloser(driver)

	source, errHTTPFs := httpfs.New(http.FS(migrations), "migrations")
	if errHTTPFs != nil {
		return errors.Wrap(errHTTPFs, "Failed to create httpfs for migrations")
	}

	migrator, errMigrateInstance := migrate.NewWithInstance("iofs", source, "pgx", driver)
	if errMigrateInstance != nil {
		return errors.Wrapf(errMigrateInstance, "Failed to create migrator")
	}

	if errUp := migrator.Up(); errUp != nil {
		return errors.Wrap(errUp, "Failed to migrate up")
	}

	return nil
}

type timeStamped struct {
	UpdatedOn time.Time `json:"-"`
	CreatedOn time.Time `json:"created_on"`
}

type playerNameRecord struct {
	NameID      int64         `json:"name_id"`
	SteamID     steamid.SID64 `json:"steam_id"`
	PersonaName string        `json:"persona_name"`
	CreatedOn   time.Time     `json:"created_on"`
}

type playerAvatarRecord struct {
	AvatarID   int64         `json:"avatar_id"`
	SteamID    steamid.SID64 `json:"steam_id"`
	AvatarHash string        `json:"avatar_hash"`
	CreatedOn  time.Time     `json:"created_on"`
}

type playerVanityRecord struct {
	VanityID  int64         `json:"vanity_id"`
	SteamID   steamid.SID64 `json:"steam_id"`
	Vanity    string        `json:"vanity"`
	CreatedOn time.Time     `json:"created_on"`
}

type econBanState int

const (
	econBanNone econBanState = iota
	econBanProbation
	econBanBanned
)

type playerRecord struct {
	SteamID                  steamid.SID64            `json:"steam_id"`
	CommunityVisibilityState steamweb.VisibilityState `json:"community_visibility_state"`
	ProfileState             steamweb.ProfileState    `json:"profile_state"`
	PersonaName              string                   `json:"persona_name"`
	Vanity                   string                   `json:"vanity"`
	AvatarHash               string                   `json:"avatar_hash"`
	PersonaState             steamweb.PersonaState    `json:"persona_state"`
	RealName                 string                   `json:"real_name"`
	TimeCreated              time.Time                `json:"time_created"`
	LocCountryCode           string                   `json:"loc_country_code"`
	LocStateCode             string                   `json:"loc_state_code"`
	LocCityID                int                      `json:"loc_city_id"`
	CommunityBanned          bool                     `json:"community_banned"`
	VacBanned                bool                     `json:"vac_banned"`
	LastBannedOn             time.Time                `json:"last_banned_on"`
	GameBans                 int                      `json:"game_bans"`
	EconomyBanned            econBanState             `json:"economy_banned"`
	LogsTFCount              int                      `json:"logs_tf_count"`
	UGCUpdatedOn             time.Time                `json:"ugc_updated_on"`
	RGLUpdatedOn             time.Time                `json:"rgl_updated_on"`
	ETF2LUpdatedOn           time.Time                `json:"etf2_l_updated_on"`
	LogsTFUpdatedOn          time.Time                `json:"logs_tf_updated_on"`
	isNewRecord              bool
	timeStamped
}

func (r *playerRecord) applyBans(ban steamweb.PlayerBanState) {
	r.CommunityBanned = ban.CommunityBanned
	r.VacBanned = ban.VACBanned
	r.GameBans = ban.NumberOfGameBans

	if ban.DaysSinceLastBan > 0 {
		r.LastBannedOn = time.Now().AddDate(0, 0, -ban.DaysSinceLastBan)
	}

	switch ban.EconomyBan {
	case steamweb.EconBanNone:
		r.EconomyBanned = econBanNone
	case steamweb.EconBanProbation:
		r.EconomyBanned = econBanProbation
	case steamweb.EconBanBanned:
		r.EconomyBanned = econBanBanned
	}

	r.UpdatedOn = time.Now()
}

func (r *playerRecord) applySummary(sum steamweb.PlayerSummary) {
	r.Vanity = sum.ProfileURL
	r.AvatarHash = sum.AvatarHash
	r.CommunityVisibilityState = sum.CommunityVisibilityState
	r.PersonaState = sum.PersonaState
	r.ProfileState = sum.ProfileState
	r.PersonaName = sum.PersonaName

	if sum.TimeCreated > 0 {
		r.TimeCreated = time.Unix(int64(sum.TimeCreated), 0)
	}

	r.LocCityID = sum.LocCityID
	r.LocCountryCode = sum.LocCountryCode
	r.LocStateCode = sum.LocStateCode
	r.UpdatedOn = time.Now()
}

const defaultAvatar = "fef49e7fa7e1997310d705b2a6158ff8dc1cdfeb"

func newPlayerRecord(sid64 steamid.SID64) playerRecord {
	createdOn := time.Now()

	return playerRecord{
		SteamID:                  sid64,
		CommunityVisibilityState: steamweb.VisibilityPrivate,
		ProfileState:             steamweb.ProfileStateNew,
		PersonaName:              "",
		Vanity:                   "",
		AvatarHash:               defaultAvatar,
		PersonaState:             steamweb.StateOffline,
		RealName:                 "",
		TimeCreated:              time.Time{},
		LocCountryCode:           "",
		LocStateCode:             "",
		LocCityID:                0,
		CommunityBanned:          false,
		VacBanned:                false,
		LastBannedOn:             time.Time{},
		GameBans:                 0,
		EconomyBanned:            0,
		LogsTFCount:              0,
		UGCUpdatedOn:             time.Time{},
		RGLUpdatedOn:             time.Time{},
		ETF2LUpdatedOn:           time.Time{},
		LogsTFUpdatedOn:          time.Time{},
		isNewRecord:              true,
		timeStamped: timeStamped{
			UpdatedOn: createdOn,
			CreatedOn: createdOn,
		},
	}
}

func playerNameSave(ctx context.Context, transaction pgx.Tx, record *playerRecord) error {
	query, args, errSQL := sb.
		Insert("player_names").
		Columns("steam_id", "persona_name").
		Values(record.SteamID, record.PersonaName).
		ToSql()
	if errSQL != nil {
		return dbErr(errSQL, "Failed to generate query")
	}

	if _, errName := transaction.Exec(ctx, query, args...); errName != nil {
		return dbErr(errName, "Failed to save player name")
	}

	return nil
}

func playerAvatarSave(ctx context.Context, transaction pgx.Tx, record *playerRecord) error {
	query, args, errSQL := sb.
		Insert("player_avatars").
		Columns("steam_id", "avatar_hash").
		Values(record.SteamID, record.AvatarHash).
		ToSql()
	if errSQL != nil {
		return dbErr(errSQL, "Failed to generate query")
	}

	if _, errName := transaction.Exec(ctx, query, args...); errName != nil {
		return dbErr(errName, "Failed to save player avatar")
	}

	return nil
}

func playerVanitySave(ctx context.Context, transaction pgx.Tx, record *playerRecord) error {
	if record.Vanity == "" {
		return nil
	}

	query, args, errSQL := sb.
		Insert("player_vanity").
		Columns("steam_id", "vanity").
		Values(record.SteamID, record.Vanity).
		ToSql()
	if errSQL != nil {
		return dbErr(errSQL, "Failed to generate query")
	}

	if _, errName := transaction.Exec(ctx, query, args...); errName != nil {
		return dbErr(errName, "Failed to save player vanity")
	}

	return nil
}

// nolint:dupl
func (db *pgStore) playerGetNames(ctx context.Context, sid64 steamid.SID64) ([]playerNameRecord, error) {
	query, args, errSQL := sb.
		Select("name_id", "persona_name", "created_on").
		From("player_names").
		Where(sq.Eq{"steam_id": sid64}).
		ToSql()
	if errSQL != nil {
		return nil, dbErr(errSQL, "Failed to generate query")
	}

	rows, errQuery := db.pool.Query(ctx, query, args...)
	if errQuery != nil {
		return nil, dbErr(errQuery, "Failed to get names")
	}

	defer rows.Close()

	var records []playerNameRecord

	for rows.Next() {
		record := playerNameRecord{SteamID: sid64} //nolint:exhaustruct
		if errScan := rows.Scan(&record.NameID, &record.PersonaName, &record.CreatedOn); errScan != nil {
			return nil, dbErr(errScan, "Failed to scan name record")
		}

		records = append(records, record)
	}

	return records, nil
}

func (db *pgStore) playerGetAvatars(ctx context.Context, sid64 steamid.SID64) ([]playerAvatarRecord, error) {
	query, args, errSQL := sb.
		Select("avatar_id", "avatar_hash", "created_on").
		From("player_avatars").
		Where(sq.Eq{"steam_id": sid64}).
		ToSql()
	if errSQL != nil {
		return nil, dbErr(errSQL, "Failed to generate query")
	}

	rows, errQuery := db.pool.Query(ctx, query, args...)
	if errQuery != nil {
		return nil, dbErr(errQuery, "Failed to query avatars")
	}

	defer rows.Close()

	var records []playerAvatarRecord

	for rows.Next() {
		r := playerAvatarRecord{SteamID: sid64} //nolint:exhaustruct
		if errScan := rows.Scan(&r.AvatarID, &r.AvatarHash, &r.CreatedOn); errScan != nil {
			return nil, dbErr(errScan, "Failed to scan avatar")
		}

		records = append(records, r)
	}

	return records, nil
}

func (db *pgStore) playerGetVanityNames(ctx context.Context, sid64 steamid.SID64) ([]playerVanityRecord, error) {
	query, args, errSQL := sb.
		Select("vanity_id", "vanity", "created_on").
		From("player_vanity").
		Where(sq.Eq{"steam_id": sid64}).
		ToSql()
	if errSQL != nil {
		return nil, dbErr(errSQL, "Failed to generate query")
	}

	var records []playerVanityRecord

	rows, errQuery := db.pool.Query(ctx, query, args...)
	if errQuery != nil {
		return nil, dbErr(errQuery, "Failed to query vanity names")
	}

	defer rows.Close()

	for rows.Next() {
		r := playerVanityRecord{SteamID: sid64} //nolint:exhaustruct
		if errScan := rows.Scan(&r.VanityID, &r.Vanity, &r.CreatedOn); errScan != nil {
			return nil, dbErr(errScan, "Failed to scan vanity name")
		}

		records = append(records, r)
	}

	return records, nil
}

//nolint:funlen,cyclop
func (db *pgStore) playerRecordSave(ctx context.Context, record *playerRecord) error {
	success := false

	transaction, errTx := db.pool.BeginTx(ctx, pgx.TxOptions{}) //nolint:exhaustruct
	if errTx != nil {
		return dbErr(errTx, "Failed to start transaction")
	}

	defer func() {
		if !success {
			if errRollback := transaction.Rollback(ctx); errRollback != nil {
				logger.Error("Failed to rollback player transaction", zap.Error(errRollback))
			}
		}
	}()

	if record.isNewRecord { //nolint:nestif
		query, args, errSQL := sb.
			Insert("player").
			Columns("steam_id", "community_visibility_state", "profile_state", "persona_name", "vanity",
				"avatar_hash", "persona_state", "real_name", "time_created", "loc_country_code", "loc_state_code", "loc_city_id",
				"community_banned", "vac_banned", "game_bans", "economy_banned", "logstf_count", "ugc_updated_on", "rgl_updated_on",
				"etf2l_updated_on", "logstf_updated_on", "steam_updated_on", "created_on").
			Values(record.SteamID, record.CommunityVisibilityState, record.ProfileState, record.PersonaName, record.Vanity,
				record.AvatarHash, record.PersonaState, record.RealName, record.TimeCreated, record.LocCountryCode,
				record.LocStateCode, record.LocCityID, record.CommunityBanned, record.VacBanned, record.GameBans,
				record.EconomyBanned, record.LogsTFCount, record.UGCUpdatedOn, record.RGLUpdatedOn, record.ETF2LUpdatedOn,
				record.LogsTFUpdatedOn, record.UpdatedOn, record.CreatedOn).
			ToSql()
		if errSQL != nil {
			return dbErr(errSQL, "Failed to generate query")
		}

		if _, errExec := transaction.Exec(ctx, query, args...); errExec != nil {
			return dbErr(errExec, "Failed to save player record")
		}

		record.isNewRecord = false

		if errName := playerNameSave(ctx, transaction, record); errName != nil {
			return errName
		}

		if errVanity := playerVanitySave(ctx, transaction, record); errVanity != nil {
			return errVanity
		}

		if errAvatar := playerAvatarSave(ctx, transaction, record); errAvatar != nil {
			return errAvatar
		}
	} else {
		query, args, errSQL := sb.
			Update("player").
			Set("steam_id", record.SteamID).
			Set("community_visibility_state", record.CommunityVisibilityState).
			Set("profile_state", record.ProfileState).
			Set("persona_name", record.PersonaName).
			Set("vanity", record.Vanity).
			Set("avatar_hash", record.AvatarHash).
			Set("persona_state", record.PersonaState).
			Set("real_name", record.RealName).
			Set("time_created", record.TimeCreated).
			Set("loc_country_code", record.LocCountryCode).
			Set("loc_state_code", record.LocStateCode).
			Set("loc_city_id", record.LocCityID).
			Set("community_banned", record.CommunityBanned).
			Set("vac_banned", record.VacBanned).
			Set("game_bans", record.GameBans).
			Set("economy_banned", record.EconomyBanned).
			Set("logstf_count", record.LogsTFCount).
			Set("ugc_updated_on", record.UGCUpdatedOn).
			Set("rgl_updated_on", record.RGLUpdatedOn).
			Set("etf2l_updated_on", record.ETF2LUpdatedOn).
			Set("logstf_updated_on", record.LogsTFUpdatedOn).
			Set("steam_updated_on", record.UpdatedOn).
			Where(sq.Eq{"steam_id": record.SteamID}).
			ToSql()
		if errSQL != nil {
			return dbErr(errSQL, "Failed to generate query")
		}

		if _, errExec := transaction.Exec(ctx, query, args...); errExec != nil {
			return dbErr(errExec, "Failed to update player record")
		}
	}

	if errCommit := transaction.Commit(ctx); errCommit != nil {
		return dbErr(errCommit, "Failed to commit player update transaction")
	}

	success = true

	return nil
}

// type leagueRecord struct {
//	LeagueID   int       `json:"league_id"`
//	LeagueName string    `json:"league_name"`
//	UpdatedOn  time.Time `json:"Updated_on"`
//	CreatedOn  time.Time `json:"created_on"`
//}
//
// type teamRecord struct {
//}

type sbSite struct {
	SiteID int    `json:"site_id"`
	Name   string `json:"name"`
	timeStamped
}

func newSBSite(name string) sbSite {
	createdOn := time.Now()

	return sbSite{
		SiteID: 0,
		Name:   name,
		timeStamped: timeStamped{
			UpdatedOn: createdOn,
			CreatedOn: createdOn,
		},
	}
}

type sbBanRecord struct {
	BanID       int           `json:"ban_id"`
	SiteName    string        `json:"site_name"`
	SiteID      int           `json:"site_id"`
	PersonaName string        `json:"persona_name"`
	SteamID     steamid.SID64 `json:"steam_id"`
	Reason      string        `json:"reason"`
	Duration    time.Duration `json:"duration"`
	Permanent   bool          `json:"permanent"`
	timeStamped
}

func (site sbSite) newRecord(sid64 steamid.SID64, personaName string, reason string,
	timeStamp time.Time, duration time.Duration, perm bool,
) sbBanRecord {
	return sbBanRecord{
		BanID:       0,
		SiteName:    "",
		SiteID:      site.SiteID,
		PersonaName: personaName,
		SteamID:     sid64,
		Reason:      reason,
		Duration:    duration,
		Permanent:   perm,
		timeStamped: timeStamped{
			UpdatedOn: timeStamp,
			CreatedOn: timeStamp,
		},
	}
}

func (db *pgStore) playerGetOrCreate(ctx context.Context, sid64 steamid.SID64, record *playerRecord) error {
	query, args, errSQL := sb.
		Select("steam_id", "community_visibility_state", "profile_state",
			"persona_name", "vanity", "avatar_hash", "persona_state", "real_name", "time_created", "loc_country_code",
			"loc_state_code", "loc_city_id", "community_banned", "vac_banned", "game_bans", "economy_banned",
			"logstf_count", "ugc_updated_on", "rgl_updated_on", "etf2l_updated_on", "logstf_updated_on",
			"steam_updated_on", "created_on").
		From("player").
		Where(sq.Eq{"steam_id": sid64}).
		ToSql()
	if errSQL != nil {
		return dbErr(errSQL, "Failed to generate query")
	}

	errQuery := db.pool.
		QueryRow(ctx, query, args...).
		Scan(&record.SteamID, &record.CommunityVisibilityState, &record.ProfileState, &record.PersonaName, &record.Vanity,
			&record.AvatarHash, &record.PersonaState, &record.RealName, &record.TimeCreated, &record.LocCountryCode,
			&record.LocStateCode, &record.LocCityID, &record.CommunityBanned, &record.VacBanned, &record.GameBans,
			&record.EconomyBanned, &record.LogsTFCount, &record.UGCUpdatedOn, &record.RGLUpdatedOn, &record.ETF2LUpdatedOn,
			&record.LogsTFUpdatedOn, &record.timeStamped.UpdatedOn, &record.timeStamped.CreatedOn)
	if errQuery != nil {
		wrappedErr := dbErr(errQuery, "Failed to query player")
		if errors.Is(wrappedErr, errNoRows) {
			return db.playerRecordSave(ctx, record)
		}

		return wrappedErr
	}

	record.isNewRecord = false

	return nil
}

func (db *pgStore) playerGetExpiredProfiles(ctx context.Context, limit int) ([]playerRecord, error) {
	query, args, errSQL := sb.
		Select("steam_id", "community_visibility_state", "profile_state",
			"persona_name", "vanity", "avatar_hash", "persona_state", "real_name", "time_created", "loc_country_code",
			"loc_state_code", "loc_city_id", "community_banned", "vac_banned", "game_bans", "economy_banned",
			"logstf_count", "ugc_updated_on", "rgl_updated_on", "etf2l_updated_on", "logstf_updated_on",
			"steam_updated_on", "created_on").
		From("player").
		Where("steam_updated_on < now() - interval '24 hour'").
		OrderBy("steam_updated_on desc").
		Limit(uint64(limit)).
		ToSql()
	if errSQL != nil {
		return nil, dbErr(errSQL, "Failed to generate query")
	}

	rows, errRows := db.pool.Query(ctx, query, args...)
	if errRows != nil {
		return nil, dbErr(errRows, "Failed to query expired bans")
	}

	defer rows.Close()

	var records []playerRecord

	for rows.Next() {
		var record playerRecord
		if errQuery := rows.
			Scan(&record.SteamID, &record.CommunityVisibilityState, &record.ProfileState, &record.PersonaName,
				&record.Vanity, &record.AvatarHash, &record.PersonaState, &record.RealName, &record.TimeCreated,
				&record.LocCountryCode, &record.LocStateCode, &record.LocCityID, &record.CommunityBanned,
				&record.VacBanned, &record.GameBans, &record.EconomyBanned, &record.LogsTFCount, &record.UGCUpdatedOn,
				&record.RGLUpdatedOn, &record.ETF2LUpdatedOn, &record.LogsTFUpdatedOn, &record.timeStamped.UpdatedOn,
				&record.timeStamped.CreatedOn); errQuery != nil {
			return nil, dbErr(errQuery, "Failed to scan expired ban")
		}

		records = append(records, record)
	}

	return records, nil
}

func (db *pgStore) sbSiteGetOrCreate(ctx context.Context, name string, site *sbSite) error {
	query, args, errSQL := sb.
		Select("sb_site_id", "name", "updated_on", "created_on").
		From("sb_site").
		Where(sq.Eq{"name": name}).
		ToSql()
	if errSQL != nil {
		return dbErr(errSQL, "Failed to generate query")
	}

	if errQuery := db.pool.
		QueryRow(ctx, query, args...).
		Scan(&site.SiteID, &site.Name, &site.UpdatedOn, &site.CreatedOn); errQuery != nil {
		wrappedErr := dbErr(errQuery, "Failed to query sourcebans site")
		if errors.Is(wrappedErr, errNoRows) {
			site.Name = name

			return db.sbSiteSave(ctx, site)
		}

		return wrappedErr
	}

	return nil
}

func (db *pgStore) sbSiteSave(ctx context.Context, site *sbSite) error {
	site.UpdatedOn = time.Now()

	if site.SiteID <= 0 {
		site.CreatedOn = time.Now()

		query, args, errSQL := sb.
			Insert("sb_site").
			Columns("name", "updated_on", "created_on").
			Values(site.Name, site.UpdatedOn, site.CreatedOn).
			Suffix("RETURNING sb_site_id").
			ToSql()
		if errSQL != nil {
			return dbErr(errSQL, "Failed to generate query")
		}

		if errQuery := db.pool.QueryRow(ctx, query, args...).Scan(&site.SiteID); errQuery != nil {
			return dbErr(errQuery, "Failed to save sourcebans site")
		}

		return nil
	}

	query, args, errSQL := sb.
		Update("sb_site").
		Set("name", site.Name).
		Set("updated_on", site.UpdatedOn).
		ToSql()
	if errSQL != nil {
		return dbErr(errSQL, "Failed to generate query")
	}

	if _, errQuery := db.pool.Exec(ctx, query, args...); errQuery != nil {
		return dbErr(errQuery, "Failed to update sourcebans site")
	}

	return nil
}

func (db *pgStore) sbSiteGet(ctx context.Context, siteID int, site *sbSite) error {
	query, args, errSQL := sb.
		Select("sb_site_id", "name", "updated_on", "created_on").
		From("sb_site").
		Where(sq.Eq{"sb_site_id": siteID}).
		ToSql()
	if errSQL != nil {
		return dbErr(errSQL, "Failed to generate query")
	}

	if errQuery := db.pool.QueryRow(ctx, query, args...).
		Scan(&site.SiteID, &site.Name, &site.UpdatedOn, &site.CreatedOn); errQuery != nil {
		return dbErr(errQuery, "Failed to scan sourcebans site")
	}

	return nil
}

func (db *pgStore) sbSiteDelete(ctx context.Context, siteID int) error {
	query, args, errSQL := sb.
		Delete("sb_site").
		Where(sq.Eq{"sb_site_id": siteID}).
		ToSql()
	if errSQL != nil {
		return dbErr(errSQL, "Failed to generate query")
	}

	if _, errQuery := db.pool.Exec(ctx, query, args...); errQuery != nil {
		return dbErr(errQuery, "Failed to delete sourcebans site")
	}

	return nil
}

func dbErr(err error, wrapMsg string) error {
	var pgErr *pgconn.PgError
	if ok := errors.Is(err, pgErr); ok {
		if pgErr.Code == pgerrcode.UniqueViolation {
			return errors.Wrap(errDuplicate, wrapMsg)
		}
	} else if errors.Is(err, pgx.ErrNoRows) {
		return errors.Wrap(errNoRows, "")
	}

	return errors.Wrap(err, wrapMsg)
}

func (db *pgStore) sbBanSave(ctx context.Context, record *sbBanRecord) error {
	record.UpdatedOn = time.Now()

	if record.BanID <= 0 {
		query, args, errSQL := sb.
			Insert("sb_ban").
			Columns("sb_site_id", "steam_id", "persona_name", "reason", "created_on", "duration", "permanent").
			Values(record.SiteID, record.SteamID, record.PersonaName, record.Reason, record.CreatedOn,
				record.Duration.Seconds(), record.Permanent).
			Suffix("RETURNING sb_ban_id").
			ToSql()
		if errSQL != nil {
			return dbErr(errSQL, "Failed to generate query")
		}

		if errQuery := db.pool.QueryRow(ctx, query, args...).Scan(&record.BanID); errQuery != nil {
			return dbErr(errQuery, "Failed to save ban record")
		}

		return nil
	}

	query, args, errSQL := sb.
		Update("sb_ban").
		Set("sb_site_id", record.SiteID).
		Set("steam_id", record.SteamID).
		Set("persona_name", record.PersonaName).
		Set("reason", record.Reason).
		Set("created_on", record.CreatedOn).
		Set("duration", record.Duration.Seconds()).
		Set("permanent", record.Permanent).
		ToSql()
	if errSQL != nil {
		return dbErr(errSQL, "Failed to generate query")
	}

	if _, errQuery := db.pool.Exec(ctx, query, args...); errQuery != nil {
		return dbErr(errQuery, "Failed to update sourcebans site")
	}

	return nil
}

func (db *pgStore) sbGetBansBySID(ctx context.Context, sid64 steamid.SID64) ([]sbBanRecord, error) {
	query, args, errSQL := sb.
		Select("b.sb_ban_id", "b.sb_site_id", "b.steam_id", "b.persona_name", "b.reason",
			"b.created_on", "b.duration", "b.permanent", "s.name").
		From("sb_ban b").
		LeftJoin("sb_site s ON b.sb_site_id = s.sb_site_id").
		Where(sq.Eq{"steam_id": sid64}).
		ToSql()
	if errSQL != nil {
		return nil, dbErr(errSQL, "Failed to generate query")
	}

	rows, errQuery := db.pool.Query(ctx, query, args...)
	if errQuery != nil {
		return nil, dbErr(errQuery, "Failed to query sourcebans bans")
	}

	defer rows.Close()

	var records []sbBanRecord

	for rows.Next() {
		var bRecord sbBanRecord

		var duration time.Duration

		if errScan := rows.Scan(&bRecord.BanID, &bRecord.SiteID, &bRecord.SteamID, &bRecord.PersonaName,
			&bRecord.Reason, &bRecord.CreatedOn, &duration, &bRecord.Permanent, &bRecord.SiteName); errScan != nil {
			return nil, dbErr(errScan, "Failed to scan sourcebans ban")
		}

		bRecord.Duration = duration * time.Second

		records = append(records, bRecord)
	}

	return records, nil
}
