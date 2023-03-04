package main

import (
	"github.com/jellydator/ttlcache/v3"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/leighmacdonald/steamweb"
	"log"
	"time"
)

type caches struct {
	summary *ttlcache.Cache[steamid.SID64, steamweb.PlayerSummary]
	bans    *ttlcache.Cache[steamid.SID64, steamweb.PlayerBanState]
	seasons *ttlcache.Cache[steamid.SID64, []Season]
}

func newCaches(summaryTimeout time.Duration, seasonTimeout time.Duration, bansTimeout time.Duration) caches {
	c := caches{
		summary: ttlcache.New[steamid.SID64, steamweb.PlayerSummary](
			ttlcache.WithLoader[steamid.SID64, steamweb.PlayerSummary](ttlcache.LoaderFunc[steamid.SID64, steamweb.PlayerSummary](
				func(c *ttlcache.Cache[steamid.SID64, steamweb.PlayerSummary], steamId steamid.SID64) *ttlcache.Item[steamid.SID64, steamweb.PlayerSummary] {
					ids := steamid.Collection{steamId}
					summaries, errSum := steamweb.PlayerSummaries(ids)
					if errSum != nil || len(ids) != len(summaries) {
						log.Printf("Failed to fetch summary: %v\n", errSum)
						return nil
					}

					return c.Set(steamId, summaries[0], summaryTimeout)
				},
			)),
		),
		seasons: ttlcache.New[steamid.SID64, []Season](
			ttlcache.WithLoader[steamid.SID64, []Season](ttlcache.LoaderFunc[steamid.SID64, []Season](
				func(c *ttlcache.Cache[steamid.SID64, []Season], steamId steamid.SID64) *ttlcache.Item[steamid.SID64, []Season] {
					seasons, errSum := fetchSeasons(steamId)
					if errSum != nil {
						log.Printf("Failed to fetch comp hist: %v\n", errSum)
						return nil
					}
					return c.Set(steamId, seasons, seasonTimeout)
				},
			)),
		),
		bans: ttlcache.New[steamid.SID64, steamweb.PlayerBanState](
			ttlcache.WithLoader[steamid.SID64, steamweb.PlayerBanState](ttlcache.LoaderFunc[steamid.SID64, steamweb.PlayerBanState](
				func(c *ttlcache.Cache[steamid.SID64, steamweb.PlayerBanState], steamId steamid.SID64) *ttlcache.Item[steamid.SID64, steamweb.PlayerBanState] {
					ids := steamid.Collection{steamId}
					bans, errSum := steamweb.GetPlayerBans(ids)
					if errSum != nil || len(ids) != len(bans) {
						log.Printf("Failed to fetch ban: %v\n", errSum)
						return nil
					}
					return c.Set(steamId, bans[0], bansTimeout)
				},
			)),
		)}
	return c
}

const steamCacheTimeout = time.Hour * 6
const compCacheTimeout = time.Hour * 24 * 7

var cache caches

func init() {
	cache = newCaches(steamCacheTimeout, compCacheTimeout, steamCacheTimeout)
}
