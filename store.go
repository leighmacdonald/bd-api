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
	"strings"
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
	"github.com/riverqueue/river"
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
func (db *pgStore) playerNames(ctx context.Context, sid steamid.SteamID) ([]domain.PlayerNameRecord, error) {
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
func (db *pgStore) playerAvatars(ctx context.Context, sid steamid.SteamID) ([]domain.PlayerAvatarRecord, error) {
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

func (db *pgStore) playerVanityNames(ctx context.Context, sid steamid.SteamID) ([]domain.PlayerVanityRecord, error) {
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
				"community_banned", "vac_banned", "game_bans", "economy_banned", "logstf_count", "updated_on", "created_on").
			Values(record.SteamID.Int64(), record.CommunityVisibilityState, record.ProfileState, record.PersonaName, record.Vanity,
				record.AvatarHash, record.PersonaState, record.RealName, record.TimeCreated, record.LocCountryCode,
				record.LocStateCode, record.LocCityID, record.CommunityBanned, record.VacBanned, record.GameBans,
				record.EconomyBanned, record.LogsTFCount, record.UpdatedOn, record.CreatedOn).
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
			Set("updated_on", record.UpdatedOn).
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

func (db *pgStore) playerGetOrCreate(ctx context.Context, sid steamid.SteamID, record *PlayerRecord) error {
	query, args, errSQL := sb.
		Select("community_visibility_state", "profile_state",
			"persona_name", "vanity", "avatar_hash", "persona_state", "real_name", "time_created", "loc_country_code",
			"loc_state_code", "loc_city_id", "community_banned", "vac_banned", "game_bans", "economy_banned",
			"logstf_count", "updated_on", "created_on").
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
			&record.EconomyBanned, &record.LogsTFCount, &record.TimeStamped.UpdatedOn, &record.TimeStamped.CreatedOn)
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
			"logstf_count", "updated_on", "created_on").
		From("player").
		Where("updated_on < now() - interval '24 hour'").
		OrderBy("updated_on desc").
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
				&record.VacBanned, &record.GameBans, &record.EconomyBanned, &record.LogsTFCount, &record.TimeStamped.UpdatedOn,
				&record.TimeStamped.CreatedOn); errQuery != nil {
			return nil, dbErr(errQuery, "Failed to scan expired ban")
		}

		record.SteamID = steamid.New(sid)

		records = append(records, record)
	}

	return records, nil
}

func (db *pgStore) sourcebansSiteGetOrCreate(ctx context.Context, name domain.Site, site *domain.SbSite) error {
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

			return db.sourcebansSiteSave(ctx, site)
		}

		return wrappedErr
	}

	return nil
}

func (db *pgStore) sourcebansSiteSave(ctx context.Context, site *domain.SbSite) error {
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

func (db *pgStore) sourcebansSiteGet(ctx context.Context, siteID int, site *domain.SbSite) error {
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

func (db *pgStore) sourcebansSites(ctx context.Context) ([]domain.SbSite, error) {
	query, args, errSQL := sb.
		Select("sb_site_id", "name", "updated_on", "created_on").
		From("sb_site").
		ToSql()
	if errSQL != nil {
		return nil, dbErr(errSQL, "Failed to generate query")
	}

	rows, errRows := db.pool.Query(ctx, query, args...)
	if errRows != nil {
		return nil, dbErr(errSQL, "Failed to generate rows")
	}

	defer rows.Close()

	var sites []domain.SbSite

	for rows.Next() {
		var site domain.SbSite
		if errQuery := rows.Scan(&site.SiteID, &site.Name, &site.UpdatedOn, &site.CreatedOn); errQuery != nil {
			return nil, dbErr(errQuery, "Failed to scan sourcebans site")
		}

		sites = append(sites, site)
	}

	return sites, nil
}

func (db *pgStore) sourcebansSiteDelete(ctx context.Context, siteID int) error {
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

func (db *pgStore) sourcebansBanRecordSave(ctx context.Context, record *domain.SbBanRecord) error {
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

func (db *pgStore) sourcebansRecordBySID(ctx context.Context, sids steamid.Collection) (BanRecordMap, error) {
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

func (db *pgStore) botDetectorLists(ctx context.Context) ([]domain.BDList, error) {
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

	var lists []domain.BDList
	for rows.Next() {
		var list domain.BDList
		if errScan := rows.Scan(&list.BDListID, &list.BDListName, &list.URL, &list.Game, &list.TrustWeight, &list.Deleted, &list.CreatedOn, &list.UpdatedOn); errScan != nil {
			return nil, dbErr(errScan, "failed to scan list result")
		}

		lists = append(lists, list)
	}

	return lists, nil
}

func (db *pgStore) botDetectorListByName(ctx context.Context, name string) (domain.BDList, error) {
	var list domain.BDList
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

func (db *pgStore) botDetectorListCreate(ctx context.Context, list domain.BDList) (domain.BDList, error) {
	query, args, errSQL := sb.
		Insert("bd_list").
		Columns("bd_list_name", "url", "game", "trust_weight", "deleted", "created_on", "updated_on").
		Values(list.BDListName, list.URL, list.Game, list.TrustWeight, list.Deleted, list.CreatedOn, list.UpdatedOn).
		Suffix("RETURNING bd_list_id").
		ToSql()
	if errSQL != nil {
		return domain.BDList{}, dbErr(errSQL, "Failed to build bd list create query")
	}

	if errRow := db.pool.QueryRow(ctx, query, args...).Scan(&list.BDListID); errRow != nil {
		return domain.BDList{}, dbErr(errSQL, "Failed to insert bd list create query")
	}

	return list, nil
}

func (db *pgStore) botDetectorListSave(ctx context.Context, list domain.BDList) error {
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

func (db *pgStore) botDetectorListDelete(ctx context.Context, bdListID int) error {
	query, args, errSQL := sb.Delete("bd_list").Where(sq.Eq{"bd_list_id": bdListID}).ToSql()
	if errSQL != nil {
		return dbErr(errSQL, "Failed to build bd list delete query")
	}

	if _, err := db.pool.Exec(ctx, query, args...); err != nil {
		return dbErr(err, "failed to exec delete list query")
	}

	return nil
}

func (db *pgStore) botDetectorListEntries(ctx context.Context, listID int) ([]domain.BDListEntry, error) {
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

	var results []domain.BDListEntry

	for rows.Next() {
		var (
			entry domain.BDListEntry
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

func (db *pgStore) botDetectorListEntryUpdate(ctx context.Context, entry domain.BDListEntry) error {
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

func (db *pgStore) botDetectorListEntryCreate(ctx context.Context, entry domain.BDListEntry) (domain.BDListEntry, error) {
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

func (db *pgStore) botDetectorListSearch(ctx context.Context, collection steamid.Collection, attrs []string) ([]domain.BDSearchResult, error) {
	if len(collection) == 0 {
		return []domain.BDSearchResult{}, nil
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

	var results []domain.BDSearchResult

	for rows.Next() {
		var res domain.BDSearchResult
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

func (db *pgStore) logsTFMatchCreate(ctx context.Context, match *domain.LogsTFMatch) error {
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

	if err := db.logsTFMatchInsert(ctx, transaction, match); err != nil {
		return err
	}

	if err := db.logsTFMatchPlayersInsert(ctx, transaction, match.Players); err != nil {
		return err
	}

	if err := db.logsTFMatchRoundsInsert(ctx, transaction, match.Rounds); err != nil {
		return err
	}

	if err := db.logsTFMatchMedicsInsert(ctx, transaction, match.Medics); err != nil {
		return dbErr(err, "Failed to insert logstf medic")
	}

	if err := transaction.Commit(ctx); err != nil {
		return dbErr(err, "Failed to commit")
	}

	return nil
}

func (db *pgStore) logsTFMatchGet(ctx context.Context, logID int) (*domain.LogsTFMatch, error) {
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

	players, errPlayers := db.logsTFMatchPlayers(ctx, logID)
	if errPlayers != nil {
		return nil, errPlayers
	}

	match.Players = players

	medics, errMedics := db.logsTFMatchMedics(ctx, logID)
	if errMedics != nil {
		return nil, errMedics
	}

	match.Medics = medics

	// Old format does not include rounds
	rounds, errRounds := db.logsTFMatchRounds(ctx, logID)
	if errRounds != nil && !errors.Is(errRounds, errDatabaseNoResults) {
		return nil, errRounds
	}

	if rounds == nil {
		rounds = []domain.LogsTFRound{}
	}

	match.Rounds = rounds

	return &match, nil
}

func (db *pgStore) logsTFMatchPlayers(ctx context.Context, logID int) ([]domain.LogsTFPlayer, error) {
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

	classes, errClasses := db.logsTFMatchPlayerClasses(ctx, logID)
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

func (db *pgStore) logsTFPlayerSummary(ctx context.Context, steamID steamid.SteamID) (*domain.LogsTFPlayerSummary, error) {
	const query = `
		SELECT
			count(p.log_id),
		
			coalesce(round(avg(p.kills)::numeric, 2), 0), coalesce(round(avg(p.assists)::numeric, 2), 0),
			coalesce(round(avg(p.deaths)::numeric, 2), 0), coalesce(round(avg(p.damage)::numeric, 2), 0),
			coalesce(round(avg(p.dpm)::numeric, 2), 0), coalesce(round(avg(p.kad)::numeric, 2), 0),
			coalesce(round(avg(p.kd)::numeric, 2), 0), coalesce(round(avg(p.dt)::numeric, 2), 0),
			coalesce(round(avg(p.dtm)::numeric, 2), 0), coalesce(round(avg(p.hp)::numeric, 2), 0),
			coalesce(round(avg(p.bs)::numeric, 2), 0), coalesce(round(avg(p.hs)::numeric, 2), 0),
			coalesce(round(avg(p.caps)::numeric, 2), 0), coalesce(round(avg(p.healing_taken)::numeric, 2), 0),
		
			coalesce(sum(p.kills), 0), coalesce(sum(p.assists), 0), coalesce(sum(p.deaths), 0), coalesce(sum(p.damage), 0),
			coalesce(sum(p.dt), 0), coalesce(sum(p.hp), 0), coalesce(sum(p.bs), 0), coalesce(sum(p.hs), 0),
			coalesce(sum(p.caps), 0), coalesce(sum(p.healing_taken), 0)
		FROM logstf_player p
		LEFT JOIN public.logstf l on l.log_id = p.log_id
		WHERE steam_id = $1`

	var sum domain.LogsTFPlayerSummary
	sid := steamID.Int64()
	if errScan := db.pool.QueryRow(ctx, query, sid).
		Scan(&sum.Logs,
			&sum.KillsAvg.Value, &sum.AssistsAvg.Value, &sum.DeathsAvg.Value, &sum.DamageAvg.Value,
			&sum.DPMAvg.Value, &sum.KADAvg.Value, &sum.KDAvg.Value, &sum.DamageTakenAvg.Value, &sum.DTMAvg.Value,
			&sum.HealthPacksAvg.Value, &sum.BackstabsAvg.Value, &sum.HeadshotsAvg.Value, &sum.CapsAvg.Value, &sum.HealingTakenAvg.Value,
			&sum.KillsSum, &sum.AssistsSum, &sum.DeathsSum, &sum.DamageSum,
			&sum.DamageTakenSum,
			&sum.HealthPacksSum, &sum.BackstabsSum, &sum.HeadshotsSum, &sum.CapsSum, &sum.HealingTakenSum,
		); errScan != nil {
		return nil, dbErr(errScan, "Failed to scan player")
	}

	return &sum, nil
}

func (db *pgStore) logsTFMatchRounds(ctx context.Context, logID int) ([]domain.LogsTFRound, error) {
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

func (db *pgStore) logsTFMatchPlayerClasses(ctx context.Context, logID int) ([]domain.LogsTFPlayerClass, error) {
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

func (db *pgStore) logsTFMatchMedics(ctx context.Context, logID int) ([]domain.LogsTFMedic, error) {
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

func (db *pgStore) logsTFMatchInsert(ctx context.Context, transaction pgx.Tx, match *domain.LogsTFMatch) error {
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

func (db *pgStore) logsTFMatchPlayersInsert(ctx context.Context, transaction pgx.Tx, players []domain.LogsTFPlayer) error {
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

		if err := db.logsTFMatchPlayerClassesInsert(ctx, transaction, player); err != nil {
			return dbErr(err, "Failed to insert logstf player class")
		}

		if err := db.logsTFMatchPlayerClassWeaponInsert(ctx, transaction, player); err != nil {
			return dbErr(err, "Failed to insert logstf player class weapon")
		}
	}

	return nil
}

func (db *pgStore) logsTFMatchMedicsInsert(ctx context.Context, transaction pgx.Tx, medics []domain.LogsTFMedic) error {
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

func (db *pgStore) logsTFMatchPlayerClassesInsert(ctx context.Context, transaction pgx.Tx, player domain.LogsTFPlayer) error {
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

func (db *pgStore) logsTFMatchPlayerClassWeaponInsert(ctx context.Context, transaction pgx.Tx, player domain.LogsTFPlayer) error {
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

func (db *pgStore) logsTFMatchRoundsInsert(ctx context.Context, transaction pgx.Tx, rounds []domain.LogsTFRound) error {
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

func (db *pgStore) logsTFMatchList(ctx context.Context, steamID steamid.SteamID) ([]domain.LogsTFMatchInfo, error) {
	const query = `
		SELECT l.log_id, l.title, l.map, l.format, l.views, l.duration, l.score_red, l.score_blu, l.created_on 
		FROM logstf l
		LEFT JOIN logstf_player lp on l.log_id = lp.log_id
		WHERE lp.steam_id = $1`

	rows, err := db.pool.Query(ctx, query, steamID.Int64())
	if err != nil {
		return nil, dbErr(err, "Failed to get results")
	}

	defer rows.Close()

	var matches []domain.LogsTFMatchInfo

	for rows.Next() {
		var match domain.LogsTFMatchInfo
		if errScan := rows.Scan(&match.LogID, &match.Title, &match.Map, &match.Format, &match.Views,
			&match.Duration.Duration, &match.ScoreRED, &match.ScoreBLU, &match.CreatedOn); errScan != nil {
			return nil, dbErr(errScan, "Failed to query match by id")
		}

		matches = append(matches, match)
	}

	return matches, nil
}

func (db *pgStore) logsTFLogCount(ctx context.Context, steamID steamid.Collection) (map[steamid.SteamID]int, error) {
	query, args, errQuery := sb.
		Select("count(l.log_id)", "lp.steam_id").
		From("logstf l").
		LeftJoin("logstf_player lp on l.log_id = lp.log_id").
		Where(sq.Eq{"lp.steam_id": steamID}).
		GroupBy("lp.steam_id").
		ToSql()
	if errQuery != nil {
		return nil, dbErr(errQuery, "Failed to build query")
	}

	counts := map[steamid.SteamID]int{}

	rows, errRows := db.pool.Query(ctx, query, args...)
	if errRows != nil {
		return nil, dbErr(errRows, "Failed to query logs")
	}

	defer rows.Close()

	for rows.Next() {
		var (
			count int
			sid   int64
		)
		if err := rows.Scan(&count, &sid); err != nil {
			return nil, dbErr(err, "Failed to get results")
		}

		counts[steamid.New(sid)] = count
	}

	return counts, nil
}

func (db *pgStore) logsTFNewestID(ctx context.Context) (int, error) {
	var id int
	if err := db.pool.QueryRow(ctx, "SELECT coalesce(max(log_id), 1) FROM logstf").Scan(&id); err != nil {
		return 0, dbErr(err, "failed to get max_id")
	}

	return id, nil
}

func (db *pgStore) servemeRecords(ctx context.Context) ([]domain.ServeMeRecord, error) {
	const query = `SELECT steam_id, name, reason, created_on, updated_on FROM serveme`

	rows, errRows := db.pool.Query(ctx, query)
	if errRows != nil {
		return nil, dbErr(errRows, "Failed to query serveme records")
	}

	defer rows.Close()

	var records []domain.ServeMeRecord

	for rows.Next() {
		var (
			sid    int64
			record domain.ServeMeRecord
		)
		if err := rows.Scan(&sid, &record.Name, &record.Reason, &record.CreatedOn, &record.UpdatedOn); err != nil {
			return nil, dbErr(err, "Failed to get results")
		}

		record.SteamID = steamid.New(sid)

		records = append(records, record)
	}

	return records, nil
}

func (db *pgStore) servemeUpdate(ctx context.Context, entries []domain.ServeMeRecord) error {
	transaction, errTx := db.pool.Begin(ctx)
	if errTx != nil {
		return dbErr(errTx, "Failed to create tx")
	}

	defer func() {
		if err := transaction.Rollback(ctx); err != nil {
			if !errors.Is(err, pgx.ErrTxClosed) {
				slog.Error("Failed to close tx", ErrAttr(err))
			}
		}
	}()

	if _, err := db.pool.Exec(ctx, "DELETE FROM serveme"); err != nil {
		return dbErr(err, "Failed to delete existing entries")
	}
	const query = "INSERT INTO serveme (steam_id, name, reason, created_on, updated_on) VALUES ($1, $2, $3, $4, $5)"
	for _, entry := range entries {
		if _, err := db.pool.Exec(ctx, query, entry.SteamID.Int64(), entry.Name, entry.Reason, entry.CreatedOn, entry.UpdatedOn); err != nil {
			return dbErr(err, "Failed to insert serveme record")
		}
	}

	if err := transaction.Commit(ctx); err != nil {
		return dbErr(err, "Failed to commit serveme tx")
	}

	return nil
}

func (db *pgStore) servemeRecordsSearch(ctx context.Context, collection steamid.Collection) ([]*domain.ServeMeRecord, error) {
	query, args, errQuery := sb.
		Select("steam_id", "name", "reason", "created_on", "updated_on").
		From("serveme").
		Where(sq.Eq{"steam_id": collection}).
		ToSql()
	if errQuery != nil {
		return nil, dbErr(errQuery, "Failed to create serveme query")
	}

	rows, errRows := db.pool.Query(ctx, query, args...)
	if errRows != nil {
		return nil, dbErr(errRows, "Failed to query serveme rows")
	}

	defer rows.Close()

	var records []*domain.ServeMeRecord

	for rows.Next() {
		var (
			sid    int64
			record domain.ServeMeRecord
		)

		if errScan := rows.Scan(&sid, &record.Name, &record.Reason, &record.CreatedOn, &record.UpdatedOn); errScan != nil {
			return nil, dbErr(errScan, "Failed to scan serveme record")
		}

		record.SteamID = steamid.New(sid)

		records = append(records, &record)
	}

	return records, nil
}

type siteStats struct {
	BDListEntriesCount   int                  `json:"bd_list_entries_count"`
	BDListCount          int                  `json:"bd_list_count"`
	LogsTFCount          int                  `json:"logs_tf_count"`
	LogsTFPlayerCount    int                  `json:"logs_tf_player_count"`
	PlayersCount         int                  `json:"players_count"`
	SourcebansSitesCount int                  `json:"sourcebans_sites_count"`
	SourcebansBanCount   int                  `json:"sourcebans_ban_count"`
	ServemeBanCount      int                  `json:"serveme_ban_count"`
	AvatarCount          int                  `json:"avatar_count"`
	NameCount            int                  `json:"name_count"`
	BotDetectorLists     []domain.BDListBasic `json:"bot_detector_lists"`
	SourcebansSites      []string             `json:"sourcebans_sites"`
}

func (db *pgStore) stats(ctx context.Context) (siteStats, error) {
	const query = `SELECT
		(SELECT count(bd_list_entry_id) FROM bd_list_entries) as bd_list_entries_count,
		(SELECT count(bd_list_id) FROM bd_list) as bd_list_count,
		(SELECT count(log_id) FROM logstf) as logstf_count,
		(SELECT count(distinct steam_id) FROM logstf_player) as logstf_players_count,
		(SELECT count(steam_id) FROM player) as players_count,
		(SELECT count(sb_site_id) FROM sb_site) as sb_site_count,
		(SELECT count(sb_ban_id) FROM sb_ban) as sb_ban_count,
		(SELECT count(steam_id) FROM serveme) as serveme_count,
		(SELECT count(distinct avatar_hash) FROM player_avatars) as avatar_count,
		(SELECT count(distinct persona_name) FROM player_names) as name_count`

	var stats siteStats
	if err := db.pool.QueryRow(ctx, query).
		Scan(&stats.BDListEntriesCount, &stats.BDListCount, &stats.LogsTFCount, &stats.LogsTFPlayerCount,
			&stats.PlayersCount, &stats.SourcebansSitesCount, &stats.SourcebansBanCount, &stats.ServemeBanCount,
			&stats.AvatarCount, &stats.NameCount); err != nil {
		return stats, dbErr(err, "Failed to get stats")
	}

	return stats, nil
}

func (db *pgStore) rglTeamGet(ctx context.Context, teamID int) (domain.RGLTeam, error) {
	var team domain.RGLTeam
	query, args, errQuery := sb.
		Select("team_id", "season_id", "division_id", "division_name", "team_leader", "tag", "team_name", "final_rank",
			"created_at", "updated_at").
		From("rgl_team").
		Where(sq.Eq{"team_id": teamID}).
		ToSql()
	if errQuery != nil {
		return team, dbErr(errQuery, "Failed to build query")
	}

	if err := db.pool.QueryRow(ctx, query, args...).
		Scan(&team.TeamID, &team.SeasonID, &team.DivisionID, &team.DivisionName, &team.TeamLeader, &team.Tag, &team.TeamName, &team.FinalRank,
			&team.CreatedAt, &team.UpdatedAt); err != nil {
		return team, dbErr(err, "Failed to query rgl team")
	}

	return team, nil
}

func (db *pgStore) rglTeamInsert(ctx context.Context, team domain.RGLTeam) error {
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

	if _, err := db.pool.Exec(ctx, query, args...); err != nil {
		return dbErr(err, "Failed to exec rgl team insert")
	}

	return nil
}

func (db *pgStore) rglSeasonGet(ctx context.Context, seasonID int) (domain.RGLSeason, error) {
	var season domain.RGLSeason
	query, args, errQuery := sb.
		Select("season_id", "maps", "season_name", "format_name", "region_name", "participating_teams", "matches", "created_on").
		From("rgl_season").
		Where(sq.Eq{"season_id": seasonID}).
		ToSql()
	if errQuery != nil {
		return season, dbErr(errQuery, "Failed to build query")
	}

	if err := db.pool.QueryRow(ctx, query, args...).
		Scan(&season.SeasonID, &season.Maps, &season.Name, &season.FormatName, &season.RegionName, &season.ParticipatingTeams, &season.Matches, &season.CreatedOn); err != nil {
		return season, dbErr(err, "Failed to query rgl season")
	}

	return season, nil
}

func (db *pgStore) rglSeasonInsert(ctx context.Context, season domain.RGLSeason) error {
	query, args, errQuery := sb.
		Insert("rgl_season").
		SetMap(map[string]interface{}{
			"season_id":           season.SeasonID,
			"maps":                season.Maps,
			"season_name":         season.Name,
			"format_name":         season.FormatName,
			"region_name":         season.RegionName,
			"participating_teams": season.ParticipatingTeams,
			"matches":             season.Matches,
			"created_on":          season.CreatedOn,
		}).ToSql()
	if errQuery != nil {
		return dbErr(errQuery, "Failed to insert rgl season")
	}

	if _, err := db.pool.Exec(ctx, query, args...); err != nil {
		return dbErr(err, "Failed to exec rgl season insert")
	}

	return nil
}

func (db *pgStore) rglTeamMemberInsert(ctx context.Context, member domain.RGLTeamMember) error {
	const query = `
		INSERT INTO rgl_team_member (team_id, steam_id, name, is_team_leader, joined_at, left_at) 
		VALUES ($1, $2, $3 ,$4, $5, $6)
		ON CONFLICT (team_id, steam_id)
		DO UPDATE SET left_at = $6`

	if _, err := db.pool.Exec(ctx, query, member.TeamID, member.SteamID.Int64(), member.Name, member.IsTeamLeader, member.JoinedAt, member.LeftAt); err != nil {
		return dbErr(err, "Failed to exec rgl team member insert")
	}

	return nil
}

func (db *pgStore) rglBansReplace(ctx context.Context, bans []domain.RGLBan) error {
	const query = `
		INSERT INTO rgl_ban (steam_id, alias, expires_at, created_at, reason) 
		VALUES ($1, $2, $3 ,$4, $5)`

	if _, err := db.pool.Exec(ctx, `DELETE FROM rgl_ban`); err != nil {
		return dbErr(err, "failed to cleanup rgl_ban table")
	}

	batch := &pgx.Batch{}

	for _, ban := range bans {
		record := newPlayerRecord(ban.SteamID)
		if err := db.playerGetOrCreate(ctx, ban.SteamID, &record); err != nil {
			return err
		}

		batch.Queue(query, ban.SteamID.Int64(), ban.Alias, ban.ExpiresAt, ban.CreatedAt.Truncate(time.Second), ban.Reason)
	}

	if err := db.pool.SendBatch(context.Background(), batch).Close(); err != nil {
		return dbErr(err, "Failed to send batch rgl bans")
	}

	return nil
}

func (db *pgStore) rglBansGetAll(ctx context.Context) ([]domain.RGLBan, error) {
	rows, errRows := db.pool.Query(ctx, `SELECT steam_id, alias, expires_at, created_at, reason FROM rgl_ban`)
	if errRows != nil && !errors.Is(errRows, errDatabaseNoResults) {
		return nil, dbErr(errRows, "Failed to load bans")
	}

	defer rows.Close()

	var bans []domain.RGLBan
	for rows.Next() {
		var ban domain.RGLBan
		if err := rows.Scan(&ban.SteamID, &ban.Alias, &ban.ExpiresAt, &ban.CreatedAt, &ban.Reason); err != nil {
			return nil, dbErr(err, "Failed to scan rgl bans")
		}

		bans = append(bans, ban)
	}

	return bans, nil
}

func (db *pgStore) rglMatchGet(ctx context.Context, matchID int) (domain.RGLMatch, error) {
	const query = `
		SELECT match_id, season_name, division_name, division_id, season_id, region_id, 
		       match_date, match_name, is_forfeit, winner, team_id_a, points_a, team_id_b, points_b 
		FROM rgl_match
		WHERE match_id = $1`

	var match domain.RGLMatch

	if err := db.pool.QueryRow(ctx, query, matchID).
		Scan(&match.MatchID, &match.SeasonName, &match.DivisionName, &match.DivisionID, &match.SeasonID, &match.RegionID,
			&match.MatchDate, &match.MatchName, &match.IsForfeit, &match.Winner, &match.TeamIDA, &match.PointsA,
			&match.TeamIDB, &match.PointsB); err != nil {
		return match, dbErr(err, "Failed to scan rgl match")
	}

	return match, nil
}

func (db *pgStore) rglMatchInsert(ctx context.Context, match domain.RGLMatch) error {
	query, args, errQuery := sb.Insert("rgl_match").
		SetMap(map[string]interface{}{
			"match_id":      match.MatchID,
			"season_name":   match.SeasonName,
			"division_name": match.DivisionName,
			"division_id":   match.DivisionID,
			"season_id":     match.SeasonID,
			"region_id":     match.RegionID,
			"match_date":    match.MatchDate,
			"match_name":    match.MatchName,
			"is_forfeit":    match.IsForfeit,
			"winner":        match.Winner,
			"team_id_a":     match.TeamIDA,
			"points_a":      match.PointsA,
			"team_id_b":     match.TeamIDB,
			"points_b":      match.PointsB,
		}).ToSql()
	if errQuery != nil {
		return dbErr(errQuery, "Failed to build rgl match insert query")
	}

	if _, err := db.pool.Exec(ctx, query, args...); err != nil {
		return dbErr(err, "Failed to exec rgl match insert")
	}

	return nil
}

func (db *pgStore) rglPlayerTeamHistory(ctx context.Context, steamIDs steamid.Collection) ([]domain.RGLPlayerTeamHistory, error) {
	query, args, errQuery := sb.
		Select("t.division_name", "t.team_leader", "t.tag", "t.team_name", "t.final_rank",
			"rtm.name", "rtm.is_team_leader", "rtm.joined_at", "rtm.left_at", "rtm.steam_id").
		From("rgl_team t").
		LeftJoin("rgl_team_member rtm ON t.team_id = rtm.team_id").
		Where(sq.Eq{"rtm.steam_id": steamIDs.ToInt64Slice()}).
		OrderBy("rtm.joined_at DESC").ToSql()
	if errQuery != nil {
		return nil, dbErr(errQuery, "Failed to construct rgl team history query")
	}

	rows, errRows := db.pool.Query(ctx, query, args...)
	if errRows != nil && !errors.Is(errRows, errDatabaseNoResults) {
		return nil, dbErr(errRows, "Failed to query team history")
	}

	defer rows.Close()

	teams := make([]domain.RGLPlayerTeamHistory, 0)

	for rows.Next() {
		var hist domain.RGLPlayerTeamHistory
		if err := rows.Scan(&hist.DivisionName, &hist.TeamLeader, &hist.Tag, &hist.TeamName, &hist.FinalRank,
			&hist.Name, &hist.IsTeamLeader, &hist.JoinedAt, &hist.LeftAt, &hist.SteamID); err != nil {
			return nil, dbErr(err, "Failed to scan rgl team history")
		}

		teams = append(teams, hist)
	}

	return teams, nil
}

func (db *pgStore) etf2lBansUpdate(ctx context.Context, bans []domain.ETF2LBan) error {
	const query = `
		INSERT INTO etf2l_ban (steam_id, alias, expires_at, created_at, reason) 
		VALUES ($1, $2, $3 ,$4, $5)`

	if _, err := db.pool.Exec(ctx, `DELETE FROM etf2l_ban`); err != nil {
		return dbErr(err, "Failed to delete previous bans")
	}

	batch := &pgx.Batch{}

	for _, ban := range bans {
		record := newPlayerRecord(ban.SteamID)
		if err := db.playerGetOrCreate(ctx, ban.SteamID, &record); err != nil {
			return err
		}

		batch.Queue(query, ban.SteamID.Int64(), ban.Alias, ban.ExpiresAt, ban.CreatedAt, ban.Reason)
	}

	if err := db.pool.SendBatch(context.Background(), batch).Close(); err != nil {
		return dbErr(err, "Failed to send batch etf2l bans")
	}

	return nil
}

func (db *pgStore) rglBansQuery(ctx context.Context, steamIDs steamid.Collection) ([]domain.RGLBan, error) {
	query, args, errQuery := sb.Select("steam_id", "alias", "expires_at", "created_at", "reason").
		From("rgl_ban").
		Where(sq.Eq{"steam_id": steamIDs.ToInt64Slice()}).
		ToSql()
	if errQuery != nil {
		return nil, dbErr(errQuery, "Failed to build query")
	}

	rows, errRows := db.pool.Query(ctx, query, args...)
	if errRows != nil {
		return nil, dbErr(errRows, "Failed to exec query")
	}

	defer rows.Close()

	var bans []domain.RGLBan

	for rows.Next() {
		var ban domain.RGLBan
		if errScan := rows.Scan(&ban.SteamID, &ban.Alias, &ban.ExpiresAt, &ban.CreatedAt, &ban.Reason); errScan != nil {
			return nil, dbErr(errScan, "Failed to scan ban")
		}

		bans = append(bans, ban)
	}

	return bans, nil
}

func (db *pgStore) etf2lBansQuery(ctx context.Context, steamIDs steamid.Collection) ([]domain.ETF2LBan, error) {
	query, args, errQuery := sb.
		Select("steam_id", "alias", "expires_at", "created_at", "reason").
		From("etf2l_ban").
		Where(sq.Eq{"steam_id": steamIDs}).
		ToSql()
	if errQuery != nil {
		return nil, dbErr(errQuery, "Failed to build etf2l ban query")
	}

	rows, errRows := db.pool.Query(ctx, query, args...)
	if errRows != nil {
		return nil, dbErr(errRows, "Failed to exec etf2l ban query")
	}

	defer rows.Close()

	var bans []domain.ETF2LBan

	for rows.Next() {
		var ban domain.ETF2LBan
		if errScan := rows.Scan(&ban.SteamID, &ban.Alias, &ban.ExpiresAt, &ban.CreatedAt, &ban.Reason); errScan != nil {
			return nil, dbErr(errScan, "Failed to scan etf2l ban")
		}

		bans = append(bans, ban)
	}

	return bans, nil
}

func (db *pgStore) insertJobTx(ctx context.Context, client *river.Client[pgx.Tx], job river.JobArgs, opts *river.InsertOpts) error {
	transaction, errTx := db.pool.Begin(ctx)
	if errTx != nil {
		return dbErr(errTx, "Failed to being tx")
	}

	defer func() {
		_ = transaction.Rollback(ctx)
	}()

	if _, err := client.InsertTx(ctx, transaction, job, opts); err != nil {
		slog.Error("Failed to insert rgl season followup jobs", ErrAttr(err))

		return errors.Join(err, errQueueInsert)
	}

	if err := transaction.Commit(ctx); err != nil {
		return dbErr(err, "Failed to commit jobs tx")
	}

	return nil
}

func (db *pgStore) insertJobsTx(ctx context.Context, client *river.Client[pgx.Tx], jobs []river.InsertManyParams) error {
	transaction, errTx := db.pool.Begin(ctx)
	if errTx != nil {
		return dbErr(errTx, "Failed to being tx")
	}

	defer func() {
		_ = transaction.Rollback(ctx)
	}()

	if _, err := client.InsertManyTx(ctx, transaction, jobs); err != nil {
		slog.Error("Failed to insert rgl season followup jobs", ErrAttr(err))

		return errors.Join(err, errQueueInsert)
	}

	if err := transaction.Commit(ctx); err != nil {
		return dbErr(err, "Failed to commit jobs tx")
	}

	return nil
}

func (db *pgStore) ensureSteamGame(ctx context.Context, game domain.SteamGame) error {
	const query = `
		INSERT INTO steam_game (app_id, name, img_icon_url, img_logo_url, created_on, updated_on) 
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (app_id) DO NOTHING`

	if _, err := db.pool.Exec(ctx, query, game.AppID, game.Name, game.ImgIconURL, game.ImgLogoURL, game.CreatedOn, game.UpdatedOn); err != nil {
		return dbErr(err, "Failed to ensure steam game")
	}

	return nil
}

func (db *pgStore) updateOwnedGame(ctx context.Context, game domain.SteamGameOwned) error {
	const query = `
		INSERT INTO player_owned_games (
		                                steam_id, app_id, playtime_forever_minutes, playtime_two_weeks, 
		                                has_community_visible_stats, created_on, updated_on) 
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (steam_id, app_id) DO UPDATE 
		    SET playtime_forever_minutes = $3, playtime_two_weeks = $4, has_community_visible_stats = $5, 
		        updated_on = $7`

	if _, err := db.pool.Exec(ctx, query, game.SteamID.Int64(), game.AppID, game.PlaytimeForeverMinutes, game.PlaytimeTwoWeeks, game.HasCommunityVisibleStats,
		game.CreatedOn, game.UpdatedOn); err != nil {
		return dbErr(err, "Failed to ensure steam game")
	}

	return nil
}

type OwnedGameMap map[steamid.SteamID][]domain.PlayerSteamGameOwned

func (db *pgStore) getOwnedGames(ctx context.Context, steamIDs steamid.Collection) (OwnedGameMap, error) {
	query, args, errQuery := sb.
		Select("o.steam_id", "o.app_id", "o.playtime_forever_minutes", "o.playtime_two_weeks",
			"o.has_community_visible_stats", "o.created_on", "o.updated_on", "g.name", "g.img_icon_url", "g.img_logo_url").
		From("player_owned_games o").
		LeftJoin("steam_game g USING(app_id)").
		Where(sq.Eq{"o.steam_id": steamIDs.ToInt64Slice()}).
		ToSql()
	if errQuery != nil {
		return nil, dbErr(errQuery, "Failed to build query")
	}

	owned := OwnedGameMap{}

	rows, errRows := db.pool.Query(ctx, query, args...)
	if errRows != nil {
		return nil, dbErr(errRows, "Failed to query owned games")
	}

	defer rows.Close()

	for rows.Next() {
		var (
			psgo domain.PlayerSteamGameOwned
			sid  int64
		)
		if errScan := rows.Scan(&sid, &psgo.AppID, &psgo.PlaytimeForeverMinutes, &psgo.PlaytimeTwoWeeks,
			&psgo.HasCommunityVisibleStats, &psgo.CreatedOn, &psgo.UpdatedOn, &psgo.Name, &psgo.ImgIconURL,
			&psgo.ImgLogoURL); errScan != nil {
			return nil, dbErr(errScan, "Failed to scan owned game")
		}

		psgo.SteamID = steamid.New(sid)

		if _, ok := owned[psgo.SteamID]; !ok {
			owned[psgo.SteamID] = make([]domain.PlayerSteamGameOwned, 0)
		}

		owned[psgo.SteamID] = append(owned[psgo.SteamID], psgo)
	}

	return owned, nil
}

func (db *pgStore) insertSteamServer(ctx context.Context, server domain.SteamServer) error {
	const query = `
		INSERT INTO steam_server (
		                          steam_id, addr, game_port, name, app_id, game_dir, 
		                          version, region, secure, os, tags, created_on, updated_on) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (steam_id) DO 
		    UPDATE SET 
		        addr = $2, game_port = $3, name = $4, app_id = $5, game_dir = $6, version = $7,
		        region = $8, secure = $9, os = $10, tags = $11, updated_on = $13`
	if _, err := db.pool.
		Exec(ctx, query, server.SteamID.Int64(), server.Addr, server.GamePort, server.Name, server.AppID, server.GameDir,
			server.Version, server.Region, server.Secure, server.Os, server.GameType, server.CreatedOn, server.UpdatedOn); err != nil {
		return dbErr(err, "Failed to insert server")
	}

	return nil
}

func (db *pgStore) insertSteamServersStats(ctx context.Context, stats []domain.SteamServerInfo) error {
	const query = `
		INSERT INTO steam_server_info (steam_id, time, players, bots, map_id)
		VALUES ($1, $2, $3, $4, $5)`

	batch := &pgx.Batch{}
	for _, s := range stats {
		batch.Queue(query, s.SteamID.Int64(), s.Time, s.Players, s.Bots, s.MapID)
	}

	if err := db.pool.SendBatch(ctx, batch).Close(); err != nil {
		return dbErr(err, "Failed to send server info batch")
	}

	return nil
}

func (db *pgStore) insertSteamServersCounts(ctx context.Context, stats domain.SteamServerCounts) error {
	const query = `
		INSERT INTO steam_server_counts (time, valve, community, linux, windows, vac, sdr) 
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	if _, err := db.pool.Exec(ctx, query, stats.Time, stats.Valve, stats.Community, stats.Linux, stats.Windows, stats.Vac, stats.SDR); err != nil {
		return dbErr(err, "Failed to save server counts")
	}

	return nil
}

func (db *pgStore) createMap(ctx context.Context, mapName string) (domain.Map, error) {
	const query = `INSERT INTO maps (map_name, created_on) VALUES ($1, $2) RETURNING map_id`

	mapInfo := domain.Map{
		MapName:   strings.ToLower(mapName),
		CreatedOn: time.Now(),
	}

	if err := db.pool.QueryRow(ctx, query, mapInfo.MapName, mapInfo.CreatedOn).Scan(&mapInfo.MapID); err != nil {
		if errors.Is(dbErr(err, "Duplicate"), errDatabaseUnique) {
			return db.getMap(ctx, mapInfo.MapName)
		}

		return mapInfo, dbErr(err, "Failed to create map")
	}

	return mapInfo, nil
}

func (db *pgStore) getMap(ctx context.Context, mapName string) (domain.Map, error) {
	const query = `SELECT map_id, map_name, created_on FROM maps WHERE map_name = $1`
	var mapInfo domain.Map

	if err := db.pool.QueryRow(ctx, query, mapName).
		Scan(&mapInfo.MapID, &mapInfo.MapName, &mapInfo.CreatedOn); err != nil {
		return mapInfo, dbErr(err, "Failed to query map")
	}

	return mapInfo, nil
}
