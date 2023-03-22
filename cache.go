package main

import (
	"context"
	"github.com/jellydator/ttlcache/v3"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/leighmacdonald/steamweb"
	"go.uber.org/zap"
	"time"
)

type caches struct {
	summary      *ttlcache.Cache[steamid.SID64, steamweb.PlayerSummary]
	bans         *ttlcache.Cache[steamid.SID64, steamweb.PlayerBanState]
	logsTF       *ttlcache.Cache[steamid.SID64, int64]
	friends      *ttlcache.Cache[steamid.SID64, []steamweb.Friend]
	ugcSeasons   *ttlcache.Cache[steamid.SID64, []Season]
	rglSeasons   *ttlcache.Cache[steamid.SID64, []Season]
	etf2lSeasons *ttlcache.Cache[steamid.SID64, []Season]
}

const (
	// Per cache bucket upper limit
	maxCapacity       = 100000
	steamCacheTimeout = time.Hour * 6
	compCacheTimeout  = time.Hour * 24 * 7
)

func newCaches(ctx context.Context, summaryTimeout time.Duration, seasonTimeout time.Duration, bansTimeout time.Duration) caches {
	return caches{
		summary: ttlcache.New[steamid.SID64, steamweb.PlayerSummary](
			ttlcache.WithCapacity[steamid.SID64, steamweb.PlayerSummary](maxCapacity),
			ttlcache.WithLoader[steamid.SID64, steamweb.PlayerSummary](ttlcache.LoaderFunc[steamid.SID64, steamweb.PlayerSummary](
				func(c *ttlcache.Cache[steamid.SID64, steamweb.PlayerSummary], steamId steamid.SID64) *ttlcache.Item[steamid.SID64, steamweb.PlayerSummary] {
					ids := steamid.Collection{steamId}
					summaries, errSum := steamweb.PlayerSummaries(ids)
					if errSum != nil || len(ids) != len(summaries) {
						logger.Error("Failed to fetch summary",
							zap.Error(errSum), zap.Int64("steam_id", steamId.Int64()))
						return nil
					}

					return c.Set(steamId, summaries[0], summaryTimeout)
				},
			)),
		),
		friends: ttlcache.New[steamid.SID64, []steamweb.Friend](
			ttlcache.WithCapacity[steamid.SID64, []steamweb.Friend](maxCapacity),
			ttlcache.WithLoader[steamid.SID64, []steamweb.Friend](ttlcache.LoaderFunc[steamid.SID64, []steamweb.Friend](
				func(c *ttlcache.Cache[steamid.SID64, []steamweb.Friend], steamId steamid.SID64) *ttlcache.Item[steamid.SID64, []steamweb.Friend] {
					friends, errFriends := steamweb.GetFriendList(steamId)
					if errFriends != nil {
						logger.Error("Failed to fetch friends",
							zap.Error(errFriends), zap.Int64("steam_id", steamId.Int64()))
						return nil
					}
					return c.Set(steamId, friends, summaryTimeout)
				},
			)),
		),
		ugcSeasons: ttlcache.New[steamid.SID64, []Season](
			ttlcache.WithCapacity[steamid.SID64, []Season](maxCapacity),
			ttlcache.WithLoader[steamid.SID64, []Season](ttlcache.LoaderFunc[steamid.SID64, []Season](
				func(c *ttlcache.Cache[steamid.SID64, []Season], steamId steamid.SID64) *ttlcache.Item[steamid.SID64, []Season] {
					timeout, cancel := context.WithTimeout(ctx, time.Second*10)
					defer cancel()
					seasons, errUGC := getUGC(timeout, steamId)
					if errUGC != nil {
						logger.Error("Failed to fetch ugc history",
							zap.Error(errUGC), zap.Int64("steam_id", steamId.Int64()))
						return nil
					}
					return c.Set(steamId, seasons, seasonTimeout)
				},
			)),
		),
		logsTF: ttlcache.New[steamid.SID64, int64](
			ttlcache.WithCapacity[steamid.SID64, int64](maxCapacity),
			ttlcache.WithLoader[steamid.SID64, int64](ttlcache.LoaderFunc[steamid.SID64, int64](
				func(c *ttlcache.Cache[steamid.SID64, int64], steamId steamid.SID64) *ttlcache.Item[steamid.SID64, int64] {
					timeout, cancel := context.WithTimeout(ctx, time.Second*10)
					defer cancel()
					logCount, errLogs := getLogsTF(timeout, steamId)
					if errLogs != nil {
						logger.Error("Failed to fetch logs",
							zap.Error(errLogs), zap.Int64("steam_id", steamId.Int64()))
						return nil
					}
					return c.Set(steamId, logCount, seasonTimeout)
				},
			)),
		),
		etf2lSeasons: ttlcache.New[steamid.SID64, []Season](
			ttlcache.WithCapacity[steamid.SID64, []Season](maxCapacity),
			ttlcache.WithLoader[steamid.SID64, []Season](ttlcache.LoaderFunc[steamid.SID64, []Season](
				func(c *ttlcache.Cache[steamid.SID64, []Season], steamId steamid.SID64) *ttlcache.Item[steamid.SID64, []Season] {
					timeout, cancel := context.WithTimeout(ctx, time.Second*10)
					defer cancel()
					seasons, errETF2L := getETF2L(timeout, steamId)
					if errETF2L != nil {
						logger.Error("Failed to fetch etf2l",
							zap.Error(errETF2L), zap.Int64("steam_id", steamId.Int64()))
						return nil
					}
					return c.Set(steamId, seasons, seasonTimeout)
				},
			)),
		),
		rglSeasons: ttlcache.New[steamid.SID64, []Season](
			ttlcache.WithCapacity[steamid.SID64, []Season](maxCapacity),
			ttlcache.WithLoader[steamid.SID64, []Season](ttlcache.LoaderFunc[steamid.SID64, []Season](
				func(c *ttlcache.Cache[steamid.SID64, []Season], steamId steamid.SID64) *ttlcache.Item[steamid.SID64, []Season] {
					return nil
					//timeout, cancel := context.WithTimeout(ctx, time.Second*10)
					//defer cancel()
					//seasons, errSum := getRGL(timeout, steamId)
					//if errSum != nil {
					//	log.Printf("Failed to fetch ugc hist: %v\n", errSum)
					//	return nil
					//}
					//return c.Set(steamId, seasons, seasonTimeout)
				},
			)),
		),
		bans: ttlcache.New[steamid.SID64, steamweb.PlayerBanState](
			ttlcache.WithCapacity[steamid.SID64, steamweb.PlayerBanState](maxCapacity),
			ttlcache.WithLoader[steamid.SID64, steamweb.PlayerBanState](ttlcache.LoaderFunc[steamid.SID64, steamweb.PlayerBanState](
				func(c *ttlcache.Cache[steamid.SID64, steamweb.PlayerBanState], steamId steamid.SID64) *ttlcache.Item[steamid.SID64, steamweb.PlayerBanState] {
					ids := steamid.Collection{steamId}
					bans, errBans := steamweb.GetPlayerBans(ids)
					if errBans != nil || len(ids) != len(bans) {
						logger.Error("Failed to fetch player bans",
							zap.Error(errBans), zap.Int64("steam_id", steamId.Int64()))
						return nil
					}
					return c.Set(steamId, bans[0], bansTimeout)
				},
			)),
		)}
}
