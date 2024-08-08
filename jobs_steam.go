package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/leighmacdonald/bd-api/domain"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"github.com/riverqueue/river"
	"golang.org/x/time/rate"
)

const (
	maxQueuedCount  = 100
	steamBucketSize = 200
	steamFillRate   = 1
)

var errSteamAPIResult = errors.New("failed to get data from steam api")

func NewSteamLimiter() *LimiterCustom {
	return &LimiterCustom{Limiter: rate.NewLimiter(steamFillRate, steamBucketSize)}
}

type SteamSummaryArgs struct{}

func (SteamSummaryArgs) Kind() string {
	return string(KindSteamSummary)
}

func (SteamSummaryArgs) InsertOpts() river.InsertOpts {
	return steamInsertOpts()
}

type SteamSummaryWorker struct {
	river.WorkerDefaults[SteamSummaryArgs]
	database *pgStore
	limiter  *LimiterCustom
}

func (w *SteamSummaryWorker) Timeout(_ *river.Job[SteamSummaryArgs]) time.Duration {
	return time.Second * 10
}

func (w *SteamSummaryWorker) Work(ctx context.Context, _ *river.Job[SteamSummaryArgs]) error {
	client := river.ClientFromContext[pgx.Tx](ctx)

	var expiredIDs steamid.Collection
	expiredProfiles, errProfiles := w.database.playerGetExpiredProfiles(ctx, maxQueuedCount)
	if errProfiles != nil && !errors.Is(errProfiles, errDatabaseNoResults) {
		slog.Error("Failed to fetch expired profiles", ErrAttr(errProfiles))

		return errProfiles
	}

	if len(expiredProfiles) == 0 {
		return nil
	}

	for _, profile := range expiredProfiles {
		expiredIDs = append(expiredIDs, profile.SteamID)
	}

	if _, err := getSteamSummaries(ctx, w.database, expiredIDs); err != nil {
		slog.Error("Error trying to update expired summaries")

		return err
	}

	if err := w.database.insertJobTx(ctx, client, SteamBanArgs{SteamIDs: expiredIDs}, nil); err != nil {
		return err
	}

	gameJobs := make([]river.InsertManyParams, len(expiredIDs))

	for index, sid := range expiredIDs {
		gameJobs[index] = river.InsertManyParams{
			Args: SteamGamesArgs{SteamID: sid},
		}
	}

	if err := w.database.insertJobsTx(ctx, client, gameJobs); err != nil {
		return err
	}

	return nil
}

type SteamBanArgs struct {
	SteamIDs steamid.Collection
}

func (SteamBanArgs) Kind() string {
	return string(KindSteamBan)
}

func (SteamBanArgs) InsertOpts() river.InsertOpts {
	return steamInsertOpts()
}

type SteamBanWorker struct {
	river.WorkerDefaults[SteamBanArgs]
	database *pgStore
	limiter  *LimiterCustom
}

func (w *SteamBanWorker) Work(ctx context.Context, job *river.Job[SteamBanArgs]) error {
	w.limiter.Wait(ctx)

	if _, err := getSteamBans(ctx, w.database, job.Args.SteamIDs); err != nil {
		slog.Error("Failed to refresh steam ban states", ErrAttr(err))

		return err
	}

	return nil
}

type SteamGamesArgs struct {
	SteamID steamid.SteamID `json:"steam_id"`
}

func (SteamGamesArgs) Kind() string {
	return string(KindSteamGames)
}

func (SteamGamesArgs) InsertOpts() river.InsertOpts {
	return steamInsertOpts()
}

type SteamGamesWorker struct {
	river.WorkerDefaults[SteamGamesArgs]
	database *pgStore
	limiter  *LimiterCustom
}

func (w *SteamGamesWorker) Work(ctx context.Context, job *river.Job[SteamGamesArgs]) error {
	// https://wiki.teamfortress.com/wiki/WebAPI/GetOwnedGames
	slog.Debug("Updating games", slog.String("steam_id", job.Args.SteamID.String()))

	// Back off api calls a bit so we don't error out since this endpoint does not support querying more
	// than a single steamid at a time.
	// We should be able to do a request every 1.1~ seconds non-stop with a single IP/API Key.
	// 100000 (req allowed / 24hr) / 86400 (secs/day) = ~1.15 req/sec
	w.limiter.Wait(ctx)

	if _, err := updateOwnedGames(ctx, w.database, job.Args.SteamID); err != nil {
		slog.Error("Failed to update owned games", slog.String("sid64", job.Args.SteamID.String()), ErrAttr(err))

		return err
	}

	return nil
}

type SteamServersArgs struct{}

func (SteamServersArgs) Kind() string {
	return string(KindSteamServers)
}

func (SteamServersArgs) InsertOpts() river.InsertOpts {
	opts := steamInsertOpts()
	opts.Priority = int(High)

	return opts
}

type SteamServersWorker struct {
	river.WorkerDefaults[SteamServersArgs]
	database    *pgStore
	limiter     *LimiterCustom
	serverCache map[steamid.SteamID]domain.SteamServer
	mapCache    map[string]domain.Map
}

func (w *SteamServersWorker) Work(ctx context.Context, _ *river.Job[SteamServersArgs]) error {
	var (
		stats  []domain.SteamServerInfo
		counts domain.SteamServerCounts
	)

	for _, region := range []int{0, 1, 2, 3, 4, 5, 6, 7, 255} {
		now := time.Now()
		newServers, errServers := steamweb.GetServerList(ctx, map[string]string{
			"dedicated": "1",
			"appid":     "440",
			"region":    fmt.Sprintf("%d", region),
		})
		if errServers != nil {
			slog.Error("Failed to get servers", ErrAttr(errServers))

			return errors.Join(errServers, errFetchServers)
		}

		for _, server := range newServers {
			sid := steamid.New(server.Steamid)

			_, err := w.ensureServer(ctx, sid, now, server)
			if err != nil {
				return err
			}

			mapInfo, errMapInfo := w.ensureMap(ctx, server.Map)
			if errMapInfo != nil {
				return errMapInfo
			}

			stats = append(stats, domain.SteamServerInfo{
				SteamID: sid,
				Time:    now,
				Players: server.Players,
				Bots:    server.Bots,
				MapID:   mapInfo.MapID,
			})

			if server.Os == "l" {
				counts.Linux++
			} else {
				counts.Windows++
			}

			if server.Secure {
				counts.Vac++
			}

			if strings.HasPrefix(server.Addr, "169.254") {
				counts.SDR++
			}

			if strings.HasPrefix(server.Name, "Valve Matchmaking") && server.Region == 255 {
				counts.Valve++
			} else {
				counts.Community++
			}

			counts.Time = now
		}

		time.Sleep(time.Second)
	}

	if err := w.database.insertSteamServersStats(ctx, stats); err != nil {
		return err
	}

	if err := w.database.insertSteamServersCounts(ctx, counts); err != nil {
		return err
	}

	return nil
}

func (w *SteamServersWorker) ensureServer(ctx context.Context, sid steamid.SteamID, now time.Time, server steamweb.Server) (domain.SteamServer, error) {
	// Insert new servers or update expired ones
	cacheServer, found := w.serverCache[sid]
	if found && time.Since(cacheServer.UpdatedOn) < time.Hour*12 {
		return cacheServer, nil
	}

	host, port, errParse := parseHostPort(server.Addr)
	if errParse != nil {
		return cacheServer, errParse
	}

	steamServer := domain.SteamServer{
		SteamID:    sid,
		Addr:       host,
		GamePort:   port,
		Name:       server.Name,
		AppID:      server.Appid,
		GameDir:    server.GameDir,
		Version:    server.Version,
		Region:     server.Region,
		MaxPlayers: server.MaxPlayers,
		Secure:     server.Secure,
		Os:         server.Os,
		GameType:   strings.Split(strings.ToLower(server.GameType), ","),
		TimeStamped: domain.TimeStamped{
			UpdatedOn: now,
			CreatedOn: now,
		},
	}

	if err := w.database.insertSteamServer(ctx, steamServer); err != nil {
		return steamServer, err
	}

	w.serverCache[sid] = steamServer

	return steamServer, nil
}

func (w *SteamServersWorker) ensureMap(ctx context.Context, mapName string) (domain.Map, error) {
	mapName = strings.ToLower(mapName)

	mapInfo, mapFound := w.mapCache[mapName]
	if mapFound {
		return mapInfo, nil
	}

	newMap, errMI := w.database.createMap(ctx, mapName)
	if errMI != nil {
		return mapInfo, errMI
	}

	w.mapCache[mapName] = newMap

	return newMap, nil
}
