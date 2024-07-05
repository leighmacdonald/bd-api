package main

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/golang-migrate/migrate/v4"
	pgxMigrate "github.com/golang-migrate/migrate/v4/database/pgx"
	"github.com/golang-migrate/migrate/v4/source/httpfs"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/leighmacdonald/bd-api/domain"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/steamweb/v2"
)

var (
	// Use $ for pg based queries.
	sb = sq.StatementBuilder.PlaceholderFormat(sq.Dollar) //nolint:gochecknoglobals,varnamelen
	//go:embed migrations
	migrations embed.FS

	errDatabasePing      = errors.New("failed to ping database")
	errDatabaseDSN       = errors.New("failed to parse database dsn")
	errDatabaseMigrate   = errors.New("failed to migrate database")
	errDatabaseConnect   = errors.New("failed to connect to database")
	errDatabaseUnique    = errors.New("unique record violation")
	errDatabaseNoResults = errors.New("no results")
	errDatabaseQuery     = errors.New("query error")
	errDatabaseInvalidID = errors.New("invalid id")
)

func newStore(ctx context.Context, dsn string) (*pgStore, error) {
	log := slog.With(slog.String("name", "db"))
	cfg, errConfig := pgxpool.ParseConfig(dsn)

	if errConfig != nil {
		return nil, errors.Join(errConfig, errDatabaseDSN)
	}

	database := pgStore{
		log:  log,
		dsn:  dsn,
		pool: nil,
	}

	if errMigrate := database.migrate(); errMigrate != nil {
		if errMigrate.Error() == "no change" {
			database.log.Debug("Migration at latest version")
		} else {
			return nil, errors.Join(errMigrate, errDatabaseMigrate)
		}
	} else {
		database.log.Debug("Migration completed successfully")
	}

	dbConn, errConnectConfig := pgxpool.NewWithConfig(ctx, cfg)
	if errConnectConfig != nil {
		return nil, errors.Join(errConnectConfig, errDatabaseConnect)
	}

	database.pool = dbConn

	return &database, nil
}

type pgStore struct {
	dsn  string
	log  *slog.Logger
	pool *pgxpool.Pool
}

// Migrate database schema.
func (db *pgStore) migrate() error {
	const stmtTimeout = 60 * time.Second

	instance, errOpen := sql.Open("pgx", db.dsn)
	if errOpen != nil {
		return errDatabaseMigrate
	}

	if err := instance.Ping(); err != nil {
		return errors.Join(err, errDatabaseMigrate)
	}

	driver, errMigrate := pgxMigrate.WithInstance(instance, &pgxMigrate.Config{ //nolint:exhaustruct
		MigrationsTable:       "_migration",
		SchemaName:            "public",
		StatementTimeout:      stmtTimeout,
		MultiStatementEnabled: true,
	})
	if errMigrate != nil {
		return errors.Join(errMigrate, errDatabaseMigrate)
	}

	defer logCloser(driver)

	source, errHTTPFs := httpfs.New(http.FS(migrations), "migrations")
	if errHTTPFs != nil {
		return errors.Join(errHTTPFs, errDatabaseMigrate)
	}

	migrator, errMigrateInstance := migrate.NewWithInstance("iofs", source, "pgx", driver)
	if errMigrateInstance != nil {
		return errors.Join(errMigrateInstance, errDatabaseMigrate)
	}

	errMigration := migrator.Up()

	if errMigration != nil && errMigration.Error() != "no change" {
		return errors.Join(errMigration, errDatabaseMigrate)
	}

	return nil
}

type PlayerRecord struct {
	domain.Player
	isNewRecord bool
}

func (r *PlayerRecord) applyBans(ban steamweb.PlayerBanState) {
	r.CommunityBanned = ban.CommunityBanned
	r.VacBanned = ban.VACBanned
	r.GameBans = ban.NumberOfGameBans

	if ban.DaysSinceLastBan > 0 {
		r.LastBannedOn = time.Now().AddDate(0, 0, -ban.DaysSinceLastBan)
	}

	switch ban.EconomyBan {
	case steamweb.EconBanNone:
		r.EconomyBanned = domain.EconBanNone
	case steamweb.EconBanProbation:
		r.EconomyBanned = domain.EconBanProbation
	case steamweb.EconBanBanned:
		r.EconomyBanned = domain.EconBanBanned
	}

	r.UpdatedOn = time.Now()
}

func (r *PlayerRecord) applySummary(sum steamweb.PlayerSummary) {
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

func newPlayerRecord(sid64 steamid.SteamID) PlayerRecord {
	createdOn := time.Now()

	return PlayerRecord{
		Player: domain.Player{
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
			TimeStamped: domain.TimeStamped{
				UpdatedOn: createdOn,
				CreatedOn: createdOn,
			},
		},
		isNewRecord: true,
	}
}

func playerNameSave(ctx context.Context, transaction pgx.Tx, record *PlayerRecord) error {
	query, args, errSQL := sb.
		Insert("player_names").
		Columns("steam_id", "persona_name").
		Values(record.SteamID.Int64(), record.PersonaName).
		ToSql()
	if errSQL != nil {
		return dbErr(errSQL, "Failed to generate query")
	}

	if _, errName := transaction.Exec(ctx, query, args...); errName != nil {
		return dbErr(errName, "Failed to save player name")
	}

	return nil
}

func playerAvatarSave(ctx context.Context, transaction pgx.Tx, record *PlayerRecord) error {
	query, args, errSQL := sb.
		Insert("player_avatars").
		Columns("steam_id", "avatar_hash").
		Values(record.SteamID.Int64(), record.AvatarHash).
		ToSql()
	if errSQL != nil {
		return dbErr(errSQL, "Failed to generate query")
	}

	if _, errName := transaction.Exec(ctx, query, args...); errName != nil {
		return dbErr(errName, "Failed to save player avatar")
	}

	return nil
}

func playerVanitySave(ctx context.Context, transaction pgx.Tx, record *PlayerRecord) error {
	if record.Vanity == "" {
		return nil
	}

	query, args, errSQL := sb.
		Insert("player_vanity").
		Columns("steam_id", "vanity").
		Values(record.SteamID.Int64(), record.Vanity).
		ToSql()
	if errSQL != nil {
		return dbErr(errSQL, "Failed to generate query")
	}

	if _, errName := transaction.Exec(ctx, query, args...); errName != nil {
		return dbErr(errName, "Failed to save player vanity")
	}

	return nil
}

//nolint:dupl
func (db *pgStore) playerGetNames(ctx context.Context, sid steamid.SteamID) ([]domain.PlayerNameRecord, error) {
	query, args, errSQL := sb.
		Select("name_id", "persona_name", "created_on").
		From("player_names").
		Where(sq.Eq{"steam_id": sid.Int64()}).
		ToSql()
	if errSQL != nil {
		return nil, dbErr(errSQL, "Failed to generate query")
	}

	rows, errQuery := db.pool.Query(ctx, query, args...)
	if errQuery != nil {
		return nil, dbErr(errQuery, "Failed to get names")
	}

	defer rows.Close()

	var records []domain.PlayerNameRecord

	for rows.Next() {
		record := domain.PlayerNameRecord{SteamID: sid} //nolint:exhaustruct
		if errScan := rows.Scan(&record.NameID, &record.PersonaName, &record.CreatedOn); errScan != nil {
			return nil, dbErr(errScan, "Failed to scan name record")
		}

		records = append(records, record)
	}

	return records, nil
}

//nolint:dupl
func (db *pgStore) playerGetAvatars(ctx context.Context, sid steamid.SteamID) ([]domain.PlayerAvatarRecord, error) {
	query, args, errSQL := sb.
		Select("avatar_id", "avatar_hash", "created_on").
		From("player_avatars").
		Where(sq.Eq{"steam_id": sid.Int64()}).
		ToSql()
	if errSQL != nil {
		return nil, dbErr(errSQL, "Failed to generate query")
	}

	rows, errQuery := db.pool.Query(ctx, query, args...)
	if errQuery != nil {
		return nil, dbErr(errQuery, "Failed to query avatars")
	}

	defer rows.Close()

	var records []domain.PlayerAvatarRecord

	for rows.Next() {
		r := domain.PlayerAvatarRecord{SteamID: sid} //nolint:exhaustruct
		if errScan := rows.Scan(&r.AvatarID, &r.AvatarHash, &r.CreatedOn); errScan != nil {
			return nil, dbErr(errScan, "Failed to scan avatar")
		}

		records = append(records, r)
	}

	return records, nil
}

func (db *pgStore) playerGetVanityNames(ctx context.Context, sid steamid.SteamID) ([]domain.PlayerVanityRecord, error) {
	query, args, errSQL := sb.
		Select("vanity_id", "vanity", "created_on").
		From("player_vanity").
		Where(sq.Eq{"steam_id": sid.Int64()}).
		ToSql()
	if errSQL != nil {
		return nil, dbErr(errSQL, "Failed to generate query")
	}

	var records []domain.PlayerVanityRecord

	rows, errQuery := db.pool.Query(ctx, query, args...)
	if errQuery != nil {
		return nil, dbErr(errQuery, "Failed to query vanity names")
	}

	defer rows.Close()

	for rows.Next() {
		r := domain.PlayerVanityRecord{SteamID: sid} //nolint:exhaustruct
		if errScan := rows.Scan(&r.VanityID, &r.Vanity, &r.CreatedOn); errScan != nil {
			return nil, dbErr(errScan, "Failed to scan vanity name")
		}

		records = append(records, r)
	}

	return records, nil
}

func (db *pgStore) playerRecordSave(ctx context.Context, record *PlayerRecord) error {
	success := false

	transaction, errTx := db.pool.Begin(ctx) //nolint:exhaustruct
	if errTx != nil {
		return dbErr(errTx, "Failed to start transaction")
	}

	defer func() {
		if !success {
			if errRollback := transaction.Rollback(ctx); errRollback != nil {
				db.log.Error("Failed to rollback player transaction", ErrAttr(errRollback))
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
			Values(record.SteamID.Int64(), record.CommunityVisibilityState, record.ProfileState, record.PersonaName, record.Vanity,
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
			Set("steam_id", record.SteamID.Int64()).
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

func NewSBSite(name domain.Site) domain.SbSite {
	createdOn := time.Now()

	return domain.SbSite{
		SiteID: 0,
		Name:   name,
		TimeStamped: domain.TimeStamped{
			UpdatedOn: createdOn,
			CreatedOn: createdOn,
		},
	}
}

func newRecord(site domain.SbSite, sid64 steamid.SteamID, personaName string, reason string,
	timeStamp time.Time, duration time.Duration, perm bool,
) domain.SbBanRecord {
	return domain.SbBanRecord{
		BanID:       0,
		SiteName:    site.Name,
		SiteID:      site.SiteID,
		PersonaName: personaName,
		SteamID:     sid64,
		Reason:      reason,
		Duration:    duration,
		Permanent:   perm,
		TimeStamped: domain.TimeStamped{
			UpdatedOn: timeStamp,
			CreatedOn: timeStamp,
		},
	}
}

func (db *pgStore) playerGetOrCreate(ctx context.Context, sid steamid.SteamID, record *PlayerRecord) error {
	query, args, errSQL := sb.
		Select("community_visibility_state", "profile_state",
			"persona_name", "vanity", "avatar_hash", "persona_state", "real_name", "time_created", "loc_country_code",
			"loc_state_code", "loc_city_id", "community_banned", "vac_banned", "game_bans", "economy_banned",
			"logstf_count", "ugc_updated_on", "rgl_updated_on", "etf2l_updated_on", "logstf_updated_on",
			"steam_updated_on", "created_on").
		From("player").
		Where(sq.Eq{"steam_id": sid.Int64()}).
		ToSql()
	if errSQL != nil {
		return dbErr(errSQL, "Failed to generate query")
	}

	errQuery := db.pool.
		QueryRow(ctx, query, args...).
		Scan(&record.CommunityVisibilityState, &record.ProfileState, &record.PersonaName, &record.Vanity,
			&record.AvatarHash, &record.PersonaState, &record.RealName, &record.TimeCreated, &record.LocCountryCode,
			&record.LocStateCode, &record.LocCityID, &record.CommunityBanned, &record.VacBanned, &record.GameBans,
			&record.EconomyBanned, &record.LogsTFCount, &record.UGCUpdatedOn, &record.RGLUpdatedOn, &record.ETF2LUpdatedOn,
			&record.LogsTFUpdatedOn, &record.TimeStamped.UpdatedOn, &record.TimeStamped.CreatedOn)
	if errQuery != nil {
		wrappedErr := dbErr(errQuery, "Failed to query player")
		if errors.Is(wrappedErr, errDatabaseNoResults) {
			return db.playerRecordSave(ctx, record)
		}

		return wrappedErr
	}

	record.SteamID = sid
	record.isNewRecord = false

	return nil
}

func (db *pgStore) playerGetExpiredProfiles(ctx context.Context, limit int) ([]PlayerRecord, error) {
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

	var records []PlayerRecord

	for rows.Next() {
		var (
			record PlayerRecord
			sid    int64
		)
		if errQuery := rows.
			Scan(&sid, &record.CommunityVisibilityState, &record.ProfileState, &record.PersonaName,
				&record.Vanity, &record.AvatarHash, &record.PersonaState, &record.RealName, &record.TimeCreated,
				&record.LocCountryCode, &record.LocStateCode, &record.LocCityID, &record.CommunityBanned,
				&record.VacBanned, &record.GameBans, &record.EconomyBanned, &record.LogsTFCount, &record.UGCUpdatedOn,
				&record.RGLUpdatedOn, &record.ETF2LUpdatedOn, &record.LogsTFUpdatedOn, &record.TimeStamped.UpdatedOn,
				&record.TimeStamped.CreatedOn); errQuery != nil {
			return nil, dbErr(errQuery, "Failed to scan expired ban")
		}

		record.SteamID = steamid.New(sid)

		records = append(records, record)
	}

	return records, nil
}

func (db *pgStore) sbSiteGetOrCreate(ctx context.Context, name domain.Site, site *domain.SbSite) error {
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
		if errors.Is(wrappedErr, errDatabaseNoResults) {
			site.Name = name

			return db.sbSiteSave(ctx, site)
		}

		return wrappedErr
	}

	return nil
}

func (db *pgStore) sbSiteSave(ctx context.Context, site *domain.SbSite) error {
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

func (db *pgStore) sbSiteGet(ctx context.Context, siteID int, site *domain.SbSite) error {
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

func (db *pgStore) sbBanSave(ctx context.Context, record *domain.SbBanRecord) error {
	record.UpdatedOn = time.Now()

	if record.BanID <= 0 {
		query, args, errSQL := sb.
			Insert("sb_ban").
			Columns("sb_site_id", "steam_id", "persona_name", "reason", "created_on", "duration", "permanent").
			Values(record.SiteID, record.SteamID.Int64(), record.PersonaName, record.Reason, record.CreatedOn,
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
		Set("steam_id", record.SteamID.Int64()).
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

// Turn the saved usec back into seconds.
const storeDurationSecondMulti = int64(time.Second)

type BanRecordMap map[string][]domain.SbBanRecord

func (db *pgStore) sbGetBansBySID(ctx context.Context, sids steamid.Collection) (BanRecordMap, error) {
	ids := make([]int64, len(sids))
	for idx := range sids {
		ids[idx] = sids[idx].Int64()
	}

	query, args, errSQL := sb.
		Select("b.sb_ban_id", "b.sb_site_id", "b.steam_id", "b.persona_name", "b.reason",
			"b.created_on", "b.duration", "b.permanent", "s.name").
		From("sb_ban b").
		LeftJoin("sb_site s ON b.sb_site_id = s.sb_site_id").
		Where(sq.Eq{"steam_id": ids}).
		ToSql()
	if errSQL != nil {
		return nil, dbErr(errSQL, "Failed to generate query")
	}

	rows, errQuery := db.pool.Query(ctx, query, args...)
	if errQuery != nil {
		return nil, dbErr(errQuery, "Failed to query sourcebans bans")
	}

	defer rows.Close()

	records := BanRecordMap{}

	for _, sid := range sids {
		records[sid.String()] = []domain.SbBanRecord{}
	}

	for rows.Next() {
		var (
			bRecord  domain.SbBanRecord
			duration int64
			sid      int64
		)
		if errScan := rows.Scan(&bRecord.BanID, &bRecord.SiteID, &sid, &bRecord.PersonaName,
			&bRecord.Reason, &bRecord.CreatedOn, &duration, &bRecord.Permanent, &bRecord.SiteName); errScan != nil {
			return nil, dbErr(errScan, "Failed to scan sourcebans ban")
		}

		bRecord.SteamID = steamid.New(sid)
		bRecord.Duration = time.Duration(duration * storeDurationSecondMulti)

		records[bRecord.SteamID.String()] = append(records[bRecord.SteamID.String()], bRecord)
	}

	if rows.Err() != nil {
		return nil, errors.Join(rows.Err(), errDatabaseQuery)
	}

	return records, nil
}

func (db *pgStore) bdLists(ctx context.Context) ([]BDList, error) {
	query, args, errSQL := sb.
		Select("bd_list_id", "bd_list_name", "url", "game", "trust_weight", "deleted", "created_on", "updated_on").
		From("bd_list").
		ToSql()
	if errSQL != nil {
		return nil, dbErr(errSQL, "Failed to build bd list query")
	}

	rows, errQuery := db.pool.Query(ctx, query, args...)
	if errQuery != nil {
		return nil, dbErr(errQuery, "failed to query lists")
	}

	defer rows.Close()

	var lists []BDList
	for rows.Next() {
		var list BDList
		if errScan := rows.Scan(&list.BDListID, &list.BDListName, &list.URL, &list.Game, &list.TrustWeight, &list.Deleted, &list.CreatedOn, &list.UpdatedOn); errScan != nil {
			return nil, dbErr(errScan, "failed to scan list result")
		}

		lists = append(lists, list)
	}

	return lists, nil
}

func (db *pgStore) bdListByName(ctx context.Context, name string) (BDList, error) {
	var list BDList
	query, args, errSQL := sb.
		Select("bd_list_id", "bd_list_name", "url", "game", "trust_weight", "deleted", "created_on", "updated_on").
		From("bd_list").
		Where(sq.Eq{"bd_list_name": name}).
		ToSql()
	if errSQL != nil {
		return list, dbErr(errSQL, "Failed to build bd list query")
	}

	if errQuery := db.pool.QueryRow(ctx, query, args...).
		Scan(&list.BDListID, &list.BDListName, &list.URL, &list.Game, &list.TrustWeight, &list.Deleted, &list.CreatedOn, &list.UpdatedOn); errQuery != nil {
		return list, dbErr(errQuery, "failed to scan list result")
	}

	return list, nil
}

func (db *pgStore) bdListCreate(ctx context.Context, list BDList) (BDList, error) {
	query, args, errSQL := sb.
		Insert("bd_list").
		Columns("bd_list_name", "url", "game", "trust_weight", "deleted", "created_on", "updated_on").
		Values(list.BDListName, list.URL, list.Game, list.TrustWeight, list.Deleted, list.CreatedOn, list.UpdatedOn).
		Suffix("RETURNING bd_list_id").
		ToSql()
	if errSQL != nil {
		return BDList{}, dbErr(errSQL, "Failed to build bd list create query")
	}

	if errRow := db.pool.QueryRow(ctx, query, args...).Scan(&list.BDListID); errRow != nil {
		return BDList{}, dbErr(errSQL, "Failed to insert bd list create query")
	}

	return list, nil
}

func (db *pgStore) bdListSave(ctx context.Context, list BDList) error {
	query, args, errSQL := sb.
		Update("bd_list").
		SetMap(map[string]interface{}{
			"bd_list_name": list.BDListName,
			"url":          list.URL,
			"game":         list.Game,
			"trust_weight": list.TrustWeight,
			"deleted":      list.Deleted,
			"updated_on":   list.UpdatedOn,
		}).Where(sq.Eq{"bd_list_id": list.BDListID}).
		ToSql()
	if errSQL != nil {
		return dbErr(errSQL, "Failed to build bd list save query")
	}

	if _, errExec := db.pool.Exec(ctx, query, args...); errExec != nil {
		return dbErr(errSQL, "Failed to exec bd	list save query")
	}

	return nil
}

func (db *pgStore) bdListDelete(ctx context.Context, bdListID int) error {
	query, args, errSQL := sb.Delete("bd_list").Where(sq.Eq{"bd_list_id": bdListID}).ToSql()
	if errSQL != nil {
		return dbErr(errSQL, "Failed to build bd list delete query")
	}

	if _, err := db.pool.Exec(ctx, query, args...); err != nil {
		return dbErr(err, "failed to exec delete list query")
	}

	return nil
}

func (db *pgStore) bdListEntries(ctx context.Context, listID int) ([]BDListEntry, error) {
	query, args, errSQL := sb.
		Select("bd_list_entry_id", "bd_list_id", "steam_id", "attribute", "proof",
			"last_seen", "last_name", "deleted", "created_on", "updated_on").
		From("bd_list_entries").
		Where(sq.Eq{"bd_list_id": listID}).
		ToSql()
	if errSQL != nil {
		return nil, dbErr(errSQL, "Failed to build bd list entries query")
	}

	rows, errRows := db.pool.Query(ctx, query, args...)
	if errRows != nil {
		return nil, dbErr(errSQL, "Failed to execute bd list entries query")
	}

	defer rows.Close()

	var results []BDListEntry

	for rows.Next() {
		var (
			entry BDListEntry
			sid   int64
		)
		if errScan := rows.Scan(&entry.BDListEntryID, &entry.BDListID, &sid, &entry.Attributes, &entry.Proof, &entry.LastSeen,
			&entry.LastName, &entry.Deleted, &entry.CreatedOn, &entry.UpdatedOn); errScan != nil {
			return nil, dbErr(errSQL, "Failed to scan bd list entry result")
		}
		entry.SteamID = steamid.New(sid)
		results = append(results, entry)
	}

	return results, nil
}

func (db *pgStore) bdListEntryUpdate(ctx context.Context, entry BDListEntry) error {
	if entry.Proof == nil {
		entry.Proof = []string{}
	}
	query, args, errSQL := sb.
		Update("bd_list_entries").
		SetMap(map[string]interface{}{
			"attribute":  entry.Attributes,
			"proof":      entry.Proof,
			"last_seen":  entry.LastSeen,
			"last_name":  entry.LastName,
			"deleted":    entry.Deleted,
			"updated_on": entry.UpdatedOn,
		}).
		Where(sq.Eq{"bd_list_entry_id": entry.BDListEntryID}).
		ToSql()
	if errSQL != nil {
		return dbErr(errSQL, "Failed to build bd list entry update query")
	}

	if _, errExec := db.pool.Exec(ctx, query, args...); errExec != nil {
		return dbErr(errSQL, "Failed to execute bd list entry update query")
	}

	return nil
}

func (db *pgStore) bdListEntryCreate(ctx context.Context, entry BDListEntry) (BDListEntry, error) {
	if entry.Proof == nil {
		entry.Proof = []string{}
	}
	query, args, errSQL := sb.
		Insert("bd_list_entries").
		SetMap(map[string]interface{}{
			"bd_list_id": entry.BDListID,
			"steam_id":   entry.SteamID.Int64(),
			"attribute":  entry.Attributes,
			"proof":      entry.Proof,
			"last_seen":  entry.LastSeen,
			"last_name":  entry.LastName,
			"deleted":    entry.Deleted,
			"created_on": entry.CreatedOn,
			"updated_on": entry.UpdatedOn,
		}).
		Suffix("RETURNING bd_list_entry_id").
		ToSql()
	if errSQL != nil {
		return entry, dbErr(errSQL, "Failed to build bd list entry update query")
	}

	if errScan := db.pool.QueryRow(ctx, query, args...).Scan(&entry.BDListEntryID); errScan != nil {
		return entry, dbErr(errScan, "failed to scan list entry id")
	}

	return entry, nil
}

func (db *pgStore) bdListEntryDelete(ctx context.Context, entryID int64) error {
	if entryID <= 0 {
		return errDatabaseInvalidID
	}

	query, args, errSQL := sb.
		Delete("bd_list_entries").
		Where(sq.Eq{"bd_list_entry_id": entryID}).
		ToSql()
	if errSQL != nil {
		return dbErr(errSQL, "Failed to build bd list entry delete query")
	}

	if _, err := db.pool.Exec(ctx, query, args...); err != nil {
		return dbErr(err, "failed to execute delete list entry")
	}

	return nil
}

type BDSearchResult struct {
	ListName string      `json:"list_name"`
	Match    TF2BDPlayer `json:"match"`
}

func (db *pgStore) bdListSearch(ctx context.Context, collection steamid.Collection, attrs []string) ([]BDSearchResult, error) {
	if len(collection) == 0 {
		return []BDSearchResult{}, nil
	}

	if len(attrs) == 0 {
		attrs = []string{"cheater"}
	}
	attrs = normalizeAttrs(attrs)
	conditions := sq.And{sq.Eq{"e.steam_id": steamIDCollectionToInt64Slice(collection)}}

	if !slices.Contains(attrs, "all") {
		conditions = append(conditions, sq.Expr("e.attribute && ?", attrs))
	}

	query, args, errSQL := sb.
		Select("l.bd_list_name", "e.attribute", "e.proof", "e.last_name", "e.last_seen", "e.steam_id").
		From("bd_list l").
		LeftJoin("bd_list_entries e ON e.bd_list_id = l.bd_list_id").
		Where(conditions).
		ToSql()
	if errSQL != nil {
		return nil, dbErr(errSQL, "Failed to build bd list entry delete query")
	}

	rows, errRows := db.pool.Query(ctx, query, args...)
	if errRows != nil {
		return nil, dbErr(errRows, "failed to execute list search entry")
	}
	defer rows.Close()

	var results []BDSearchResult

	for rows.Next() {
		var res BDSearchResult
		var lastSeen time.Time
		var steamID int64
		if errScan := rows.Scan(&res.ListName, &res.Match.Attributes, &res.Match.Proof,
			&res.Match.LastSeen.PlayerName, &lastSeen, &steamID); errScan != nil {
			return nil, dbErr(errScan, "failed to scan list search result")
		}
		res.Match.LastSeen.Time = int(lastSeen.Unix())
		res.Match.Steamid = fmt.Sprintf("%d", steamID)

		results = append(results, res)
	}

	return results, nil
}

func (db *pgStore) insertLogsTF(ctx context.Context, match *domain.LogsTFMatch) error {
	// Ensure  player FK's exist
	for _, player := range match.Players {
		playerRecord := PlayerRecord{
			Player: domain.Player{
				SteamID:     player.SteamID,
				PersonaName: player.Name,
			},
			isNewRecord: true,
		}
		if err := db.playerGetOrCreate(ctx, player.SteamID, &playerRecord); err != nil {
			return err
		}
	}

	transaction, errBegin := db.pool.Begin(ctx)
	if errBegin != nil {
		return dbErr(errBegin, "Failed to start tx")
	}

	defer func() {
		if err := transaction.Rollback(ctx); err != nil {
			if !errors.Is(err, pgx.ErrTxClosed) {
				slog.Error("failed to rollback logs tx", ErrAttr(err))
			}
		}
	}()

	if err := db.insertLogsTFMatch(ctx, transaction, match); err != nil {
		return err
	}

	if err := db.insertLogsTFMatchPlayers(ctx, transaction, match.Players); err != nil {
		return err
	}

	if err := db.insertLogsTFMatchRounds(ctx, transaction, match.Rounds); err != nil {
		return err
	}

	if err := db.insertLogsTFMatchMedics(ctx, transaction, match.Medics); err != nil {
		return dbErr(err, "Failed to insert logstf medic")
	}

	if err := transaction.Commit(ctx); err != nil {
		return dbErr(err, "Failed to commit")
	}

	return nil
}

func (db *pgStore) getLogsTFMatch(ctx context.Context, logID int) (*domain.LogsTFMatch, error) {
	const query = `
		SELECT log_id, title, map, format, views, duration, score_red, score_blu, created_on 
		FROM logstf
		WHERE log_id = $1`

	var match domain.LogsTFMatch
	if err := db.pool.QueryRow(ctx, query, logID).
		Scan(&match.LogID, &match.Title, &match.Map, &match.Format, &match.Views,
			&match.Duration.Duration, &match.ScoreRED, &match.ScoreBLU, &match.CreatedOn); err != nil {
		return nil, dbErr(err, "Failed to query match by id")
	}

	players, errPlayers := db.getLogsTFMatchPlayers(ctx, logID)
	if errPlayers != nil {
		return nil, errPlayers
	}

	match.Players = players

	medics, errMedics := db.getLogsTFMatchMedics(ctx, logID)
	if errMedics != nil {
		return nil, errMedics
	}

	match.Medics = medics

	// Old format does not include rounds
	rounds, errRounds := db.getLogsTFMatchRounds(ctx, logID)
	if errRounds != nil && !errors.Is(errRounds, errDatabaseNoResults) {
		return nil, errRounds
	}

	if rounds == nil {
		rounds = []domain.LogsTFRound{}
	}

	match.Rounds = rounds

	return &match, nil
}

func (db *pgStore) getLogsTFMatchPlayers(ctx context.Context, logID int) ([]domain.LogsTFPlayer, error) {
	const query = `
		SELECT log_id, steam_id, team, name, kills, assists, deaths, damage, dpm, kad, kd, dt, dtm, hp, bs, hs, caps, healing_taken 
		FROM logstf_player
		WHERE log_id = $1`

	rows, err := db.pool.Query(ctx, query, logID)
	if err != nil {
		return nil, dbErr(err, "Failed to query players")
	}

	defer rows.Close()

	var players []domain.LogsTFPlayer

	for rows.Next() {
		var player domain.LogsTFPlayer
		if errScan := rows.Scan(&player.LogID, &player.SteamID, &player.Team, &player.Name, &player.Kills, &player.Assists, &player.Deaths, &player.Damage, &player.DPM,
			&player.KAD, &player.KD, &player.DamageTaken, &player.DTM, &player.HealthPacks, &player.Backstabs, &player.Headshots, &player.Caps, &player.HealingTaken); errScan != nil {
			return nil, dbErr(errScan, "Failed to scan player")
		}

		players = append(players, player)
	}

	classes, errClasses := db.getLogsTFMatchPlayersClass(ctx, logID)
	if errClasses != nil {
		return nil, errClasses
	}

	for _, class := range classes {
		for _, player := range players {
			if class.SteamID == player.SteamID {
				player.Classes = append(player.Classes, class)
			}
		}
	}

	return players, nil
}

func (db *pgStore) getLogsTFPlayerSummary(ctx context.Context, steamID steamid.SteamID) (*domain.LogsTFPlayerSummary, error) {
	const query = `
		SELECT
			count(p.log_id),
			sum(case when p.team = 3 AND l.score_red > l.score_blu then 1 else 0 end),
			sum(case when p.team = 4 AND l.score_blu > l.score_red then 1 else 0 end),
			
			round(avg(p.kills)::numeric, 2), round(avg(p.assists)::numeric, 2),round(avg(p.deaths)::numeric, 2), round(avg(p.damage)::numeric, 2),
			round(avg(p.dpm)::numeric, 2), round(avg(p.kad)::numeric, 2), round(avg(p.kd)::numeric, 2), round(avg(p.dt)::numeric, 2), round(avg(p.dtm)::numeric, 2),
			round(avg(p.hp)::numeric, 2), round(avg(p.bs)::numeric, 2), round(avg(p.hs)::numeric, 2), round(avg(p.caps)::numeric, 2), round(avg(p.healing_taken)::numeric, 2),
			
			sum(p.kills), sum(p.assists),sum(p.deaths), sum(p.damage),
			sum(p.dt), 
			sum(p.hp), sum(p.bs), sum(p.hs), sum(p.caps), sum(p.healing_taken)
		FROM logstf_player p
		LEFT JOIN public.logstf l on l.log_id = p.log_id
		WHERE steam_id = $1`

	var sum domain.LogsTFPlayerSummary
	sid := steamID.Int64()
	if errScan := db.pool.QueryRow(ctx, query, sid).
		Scan(&sum.Logs, &sum.Wins, &sum.Losses,
			&sum.KillsAvg, &sum.AssistsAvg, &sum.DeathsAvg, &sum.DamageAvg,
			&sum.DPMAvg, &sum.KADAvg, &sum.KDAvg, &sum.DamageTakenAvg, &sum.DTMAvg,
			&sum.HealthPacksAvg, &sum.BackstabsAvg, &sum.HeadshotsAvg, &sum.CapsAvg, &sum.HealingTakenAvg,
			&sum.KillsSum, &sum.AssistsSum, &sum.DeathsSum, &sum.DamageSum,
			&sum.DamageTakenSum,
			&sum.HealthPacksSum, &sum.BackstabsSum, &sum.HeadshotsSum, &sum.CapsSum, &sum.HealingTakenSum,
		); errScan != nil {
		return nil, dbErr(errScan, "Failed to scan player")
	}

	return &sum, nil
}

func (db *pgStore) getLogsTFMatchRounds(ctx context.Context, logID int) ([]domain.LogsTFRound, error) {
	const query = `
		SELECT log_id, round, length, score_blu, score_red, kills_blu, kills_red, ubers_blu, ubers_red, damage_blu, damage_red, midfight 
		FROM logstf_round
		WHERE log_id = $1`

	rows, err := db.pool.Query(ctx, query, logID)
	if err != nil {
		return nil, dbErr(err, "Failed to query rounds")
	}

	defer rows.Close()

	var rounds []domain.LogsTFRound

	for rows.Next() {
		var round domain.LogsTFRound
		if errScan := rows.Scan(&round.LogID, &round.Round, &round.Length.Duration, &round.ScoreBLU, &round.ScoreRED, &round.KillsBLU, &round.KillsRED,
			&round.UbersBLU, &round.UbersRED, &round.DamageBLU, &round.DamageRED, &round.MidFight); errScan != nil {
			return nil, dbErr(errScan, "Failed to scan round")
		}

		rounds = append(rounds, round)
	}

	return rounds, nil
}

func (db *pgStore) getLogsTFMatchPlayersClass(ctx context.Context, logID int) ([]domain.LogsTFPlayerClass, error) {
	const query = `
		SELECT log_id, steam_id, player_class, played, kills, assists, deaths, damage 
		FROM logstf_player_class
		WHERE log_id = $1`

	rows, err := db.pool.Query(ctx, query, logID)
	if err != nil {
		return nil, dbErr(err, "Failed to query classes")
	}

	defer rows.Close()

	var classes []domain.LogsTFPlayerClass

	for rows.Next() {
		var p domain.LogsTFPlayerClass
		if errScan := rows.Scan(&p.LogID, &p.SteamID, &p.Class, &p.Played.Duration, &p.Kills, &p.Assists, &p.Deaths, &p.Damage); errScan != nil {
			return nil, dbErr(errScan, "Failed to scan player class")
		}

		classes = append(classes, p)
	}

	return classes, nil
}

func (db *pgStore) getLogsTFMatchMedics(ctx context.Context, logID int) ([]domain.LogsTFMedic, error) {
	const query = `
		SELECT log_id, steam_id, healing, charges_kritz, charges_quickfix, charges_medigun, charges_vacc, avg_time_build, 
		       avg_time_use, near_full_death, avg_uber_len, death_after_charge, major_adv_lost, biggest_adv_lost 
		FROM logstf_medic
		WHERE log_id = $1`

	rows, err := db.pool.Query(ctx, query, logID)
	if err != nil {
		return nil, dbErr(err, "Failed to query medics")
	}

	defer rows.Close()

	var medics []domain.LogsTFMedic

	for rows.Next() {
		var medic domain.LogsTFMedic
		if errScan := rows.Scan(&medic.LogID, &medic.SteamID, &medic.Healing, &medic.ChargesKritz, &medic.ChargesQuickfix, &medic.ChargesMedigun, &medic.ChargesVacc,
			&medic.AvgTimeBuild.Duration, &medic.AvgTimeUse.Duration, &medic.NearFullDeath, &medic.AvgUberLen.Duration, &medic.DeathAfterCharge, &medic.MajorAdvLost, &medic.BiggestAdvLost.Duration); errScan != nil {
			return nil, dbErr(errScan, "Failed to scan medic")
		}

		medics = append(medics, medic)
	}

	return medics, nil
}

func (db *pgStore) insertLogsTFMatch(ctx context.Context, transaction pgx.Tx, match *domain.LogsTFMatch) error {
	matchQuery, matchArgs, errQuery := sb.Insert("logstf").
		SetMap(map[string]interface{}{
			"log_id":     match.LogID,
			"title":      match.Title,
			"map":        match.Map,
			"format":     match.Format,
			"views":      match.Views,
			"duration":   match.Duration.Duration,
			"score_red":  match.ScoreRED,
			"score_blu":  match.ScoreBLU,
			"created_on": match.CreatedOn,
		}).ToSql()
	if errQuery != nil {
		return dbErr(errQuery, "Failed to build query")
	}

	if _, err := transaction.Exec(ctx, matchQuery, matchArgs...); err != nil {
		return dbErr(err, "Failed to insert logstf table")
	}

	return nil
}

func (db *pgStore) insertLogsTFMatchPlayers(ctx context.Context, transaction pgx.Tx, players []domain.LogsTFPlayer) error {
	for _, player := range players {
		query, args, errQuery := sb.Insert("logstf_player").
			SetMap(map[string]interface{}{
				"log_id":        player.LogID,
				"steam_id":      player.SteamID,
				"team":          player.Team,
				"name":          player.Name,
				"kills":         player.Kills,
				"assists":       player.Assists,
				"deaths":        player.Deaths,
				"damage":        player.Damage,
				"dpm":           player.DPM,
				"kad":           player.KAD,
				"kd":            player.KD,
				"dt":            player.DamageTaken,
				"dtm":           player.DTM,
				"hp":            player.HealthPacks,
				"bs":            player.Backstabs,
				"hs":            player.Headshots,
				"caps":          player.Caps,
				"healing_taken": player.HealingTaken,
			}).ToSql()
		if errQuery != nil {
			return dbErr(errQuery, "Failed to build query")
		}

		if _, err := transaction.Exec(ctx, query, args...); err != nil {
			return dbErr(err, "Failed to insert logstf table")
		}

		if err := db.insertLogsTFMatchPlayerClasses(ctx, transaction, player); err != nil {
			return dbErr(err, "Failed to insert logstf player class")
		}

		if err := db.insertLogsTFMatchPlayerClassWeapon(ctx, transaction, player); err != nil {
			return dbErr(err, "Failed to insert logstf player class weapon")
		}
	}

	return nil
}

func (db *pgStore) insertLogsTFMatchMedics(ctx context.Context, transaction pgx.Tx, medics []domain.LogsTFMedic) error {
	for _, medic := range medics {
		query, args, errQuery := sb.Insert("logstf_medic").
			SetMap(map[string]interface{}{
				"log_id":             medic.LogID,
				"steam_id":           medic.SteamID,
				"healing":            medic.Healing,
				"charges_kritz":      medic.ChargesKritz,
				"charges_quickfix":   medic.ChargesQuickfix,
				"charges_medigun":    medic.ChargesMedigun,
				"charges_vacc":       medic.ChargesVacc,
				"avg_time_build":     medic.AvgTimeBuild.Duration,
				"avg_time_use":       medic.AvgTimeUse.Duration,
				"near_full_death":    medic.NearFullDeath,
				"avg_uber_len":       medic.AvgUberLen.Duration,
				"death_after_charge": medic.DeathAfterCharge,
				"major_adv_lost":     medic.MajorAdvLost,
				"biggest_adv_lost":   medic.BiggestAdvLost.Duration,
			}).ToSql()
		if errQuery != nil {
			return dbErr(errQuery, "Failed to build query")
		}

		if _, err := transaction.Exec(ctx, query, args...); err != nil {
			return dbErr(err, "Failed to insert logstf table")
		}
	}

	return nil
}

func (db *pgStore) insertLogsTFMatchPlayerClasses(ctx context.Context, transaction pgx.Tx, player domain.LogsTFPlayer) error {
	for _, class := range player.Classes {
		query, args, errQuery := sb.Insert("logstf_player_class").
			SetMap(map[string]interface{}{
				"log_id":       player.LogID,
				"steam_id":     player.SteamID,
				"player_class": class.Class,
				"played":       class.Played.Duration,
				"kills":        class.Kills,
				"assists":      class.Assists,
				"deaths":       class.Deaths,
				"damage":       class.Damage,
			}).ToSql()
		if errQuery != nil {
			return dbErr(errQuery, "Failed to build query")
		}

		if _, err := transaction.Exec(ctx, query, args...); err != nil {
			return dbErr(err, "Failed to insert logstf table")
		}
	}

	return nil
}

func (db *pgStore) insertLogsTFMatchPlayerClassWeapon(ctx context.Context, transaction pgx.Tx, player domain.LogsTFPlayer) error {
	for _, class := range player.Classes {
		for _, classWeapon := range class.Weapons {
			query, args, errQuery := sb.Insert("logstf_player_class_weapon").
				SetMap(map[string]interface{}{
					"log_id":   player.LogID,
					"steam_id": player.SteamID,
					"weapon":   classWeapon.Weapon,
					"kills":    classWeapon.Kills,
					"damage":   classWeapon.Damage,
					"accuracy": classWeapon.Accuracy,
				}).ToSql()
			if errQuery != nil {
				return dbErr(errQuery, "Failed to build query")
			}

			if _, err := transaction.Exec(ctx, query, args...); err != nil {
				return dbErr(err, "Failed to insert logstf table")
			}
		}
	}

	return nil
}

func (db *pgStore) insertLogsTFMatchRounds(ctx context.Context, transaction pgx.Tx, rounds []domain.LogsTFRound) error {
	for _, player := range rounds {
		query, args, errQuery := sb.Insert("logstf_round").
			SetMap(map[string]interface{}{
				"log_id":     player.LogID,
				"round":      player.Round,
				"length":     player.Length.Duration,
				"score_blu":  player.ScoreBLU,
				"score_red":  player.ScoreRED,
				"kills_blu":  player.KillsBLU,
				"kills_red":  player.KillsRED,
				"ubers_blu":  player.UbersBLU,
				"ubers_red":  player.UbersRED,
				"damage_blu": player.DamageBLU,
				"damage_red": player.DamageRED,
				"midfight":   player.MidFight,
			}).ToSql()
		if errQuery != nil {
			return dbErr(errQuery, "Failed to build query")
		}

		if _, err := transaction.Exec(ctx, query, args...); err != nil {
			return dbErr(err, "Failed to insert logstf table")
		}
	}

	return nil
}

func (db *pgStore) getNewestLogID(ctx context.Context) (int, error) {
	var id int
	if err := db.pool.QueryRow(ctx, "SELECT max(log_id) FROM logstf").Scan(&id); err != nil {
		return 0, dbErr(err, "failed to get max_id")
	}

	return id, nil
}

func steamIDCollectionToInt64Slice(collection steamid.Collection) []int64 {
	ids := make([]int64, len(collection))
	for idx := range collection {
		ids[idx] = collection[idx].Int64()
	}

	return ids
}

func dbErr(err error, wrapMsg string) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == pgerrcode.UniqueViolation {
			return errors.Join(err, errDatabaseUnique)
		}
	} else if errors.Is(err, pgx.ErrNoRows) {
		return errors.Join(err, errDatabaseNoResults)
	}

	return errors.Join(err, fmt.Errorf("%w: %s", errDatabaseQuery, wrapMsg))
}
