# API Reference

All endpoints that support multiple steam ids are limited to a maximum of 100 steam ids per query.

## GET /bans

Returns the current vac ban states for the requested IDs. 

This returns children with the identical data shape of the [steam web api](https://steamapi.xpaw.me/#ISteamUser/GetPlayerBans).

Example: https://bd-api.roto.lol/bans?steamids=76561197970669109,76561197992870439

```json
[
  {
    "SteamId": "76561197992870439",
    "CommunityBanned": false,
    "VACBanned": false,
    "NumberOfVACBans": 0,
    "DaysSinceLastBan": 0,
    "NumberOfGameBans": 0,
    "EconomyBan": "none"
  },
  {
    "SteamId": "76561197970669109",
    "CommunityBanned": false,
    "VACBanned": false,
    "NumberOfVACBans": 0,
    "DaysSinceLastBan": 0,
    "NumberOfGameBans": 0,
    "EconomyBan": "none"
  }
]
```

## GET /friends

Fetch friends lists for the requested steam ids. Depending on the profile visibility states, this will return a empty
list when users set their profiles to private.

This returns a map with children using the identical data shape of the [steam web api](https://steamapi.xpaw.me/#IFriendsListService).

Example: https://bd-api.roto.lol/friends?steamids=76561197970669109,76561197992870439

```json
{
    "76561197970669109": [
        {
            "steamid": "76561197960265749",
            "relationship": "friend",
            "friend_since": 1449274040
        },
        {
            "steamid": "76561197960269040",
            "relationship": "friend",
            "friend_since": 1449272149
        },
        {
            "steamid": "76561197960274521",
            "relationship": "friend",
            "friend_since": 1317092444
        }
    ],
    "76561197992870439": []
}
```

## GET /summary

Fetch player summaries for all requested steam ids. 

Returns a list of player summaries as children. Mirrors data shape of [steam web api](https://steamapi.xpaw.me/#ISteamUser/GetPlayerSummaries).. 

Example: https://bd-api.roto.lol/summary?steamids=76561197970669109,76561197992870439

```json
[
  {
    "steamid": "76561197992870439",
    "communityvisibilitystate": 3,
    "profilestate": 1,
    "personaname": "camp3r",
    "profileurl": "https://steamcommunity.com/id/camp3r101/",
    "avatar": "https://avatars.steamstatic.com/6df07a18bd18e0a213958fe6f9109a360da5cd84.jpg",
    "avatarmedium": "https://avatars.steamstatic.com/6df07a18bd18e0a213958fe6f9109a360da5cd84_medium.jpg",
    "avatarfull": "https://avatars.steamstatic.com/6df07a18bd18e0a213958fe6f9109a360da5cd84_full.jpg",
    "avatarhash": "6df07a18bd18e0a213958fe6f9109a360da5cd84",
    "personastate": 1,
    "realname": "camp3r#0001",
    "primaryclanid": "103582791465096569",
    "timecreated": 1190868909,
    "personastateflags": 0,
    "loccountrycode": "US",
    "locstatecode": "",
    "loccityid": 0,
    "lastlogoff": 1688774660,
    "commentpermission": 1
  },
  {
    "steamid": "76561197970669109",
    "communityvisibilitystate": 3,
    "profilestate": 1,
    "personaname": "FROYO b4nny",
    "profileurl": "https://steamcommunity.com/id/b4nny/",
    "avatar": "https://avatars.steamstatic.com/cd78b56fcb7cc9f74ae30b5b2add073f87bf7fdb.jpg",
    "avatarmedium": "https://avatars.steamstatic.com/cd78b56fcb7cc9f74ae30b5b2add073f87bf7fdb_medium.jpg",
    "avatarfull": "https://avatars.steamstatic.com/cd78b56fcb7cc9f74ae30b5b2add073f87bf7fdb_full.jpg",
    "avatarhash": "cd78b56fcb7cc9f74ae30b5b2add073f87bf7fdb",
    "personastate": 1,
    "realname": "Grant Vincent",
    "primaryclanid": "103582791436587036",
    "timecreated": 1100741621,
    "personastateflags": 0,
    "loccountrycode": "US",
    "locstatecode": "WA",
    "loccityid": 3961,
    "lastlogoff": 0,
    "commentpermission": 1
  }
]
```
## GET /profile

Profile is a higher level "meta" object that combines the following different data sources into a single object.

- Steam Player Summary
- Steam Vac States
- Steam Friends
- Competitive History RGL/UGC/ETF2L (soon)
- Sourcebans History
- LogsTF counts
- Bot Detector entries

Example: https://bd-api.roto.lol/profile?steamids=76561197970669109,76561197992870439

```json
[
  {
    "summary": {
      "steamid": "76561199234205416",
      "communityvisibilitystate": 3,
      "profilestate": 1,
      "personaname": "ìº”ë””",
      "profileurl": "https://steamcommunity.com/profiles/76561199234205416/",
      "avatar": "https://avatars.steamstatic.com/cc84710808071fa4ba4b94c3011a9576b89f6714.jpg",
      "avatarmedium": "https://avatars.steamstatic.com/cc84710808071fa4ba4b94c3011a9576b89f6714_medium.jpg",
      "avatarfull": "https://avatars.steamstatic.com/cc84710808071fa4ba4b94c3011a9576b89f6714_full.jpg",
      "avatarhash": "cc84710808071fa4ba4b94c3011a9576b89f6714",
      "personastate": 0,
      "realname": "",
      "primaryclanid": "103582791429521408",
      "timecreated": 1641867135,
      "personastateflags": 0,
      "loccountrycode": "KR",
      "locstatecode": "",
      "loccityid": 0,
      "lastlogoff": 0,
      "commentpermission": 0
    },
    "ban_state": {
      "steam_id": "76561199234205416",
      "community_banned": true,
      "vac_banned": false,
      "number_of_vac_bans": 0,
      "days_since_last_ban": 18,
      "number_of_game_bans": 1,
      "economy_ban": "none"
    },
    "source_bans": [
      {
        "ban_id": 39085,
        "site_name": "ugc",
        "site_id": 26,
        "persona_name": "U",
        "steam_id": "76561199234205416",
        "reason": "SMAC 0.8.6.3: ConVar sv_cheats violation",
        "duration": 0,
        "permanent": true,
        "created_on": "2023-06-17T07:47:34Z"
      },
      {
        "ban_id": 132632,
        "site_name": "loos",
        "site_id": 12,
        "persona_name": "winter",
        "steam_id": "76561199234205416",
        "reason": "[StAC] Banned for pSilent after 10 detections",
        "duration": 0,
        "permanent": true,
        "created_on": "2023-01-14T22:34:50Z"
      }
    ],
    "serve_me": {
      "steam_id": "76561199234205416",
      "name": "xxx",
      "reason": "match invader",
      "deleted": false,
      "created_on": "2024-07-18T01:36:27.686317-06:00"
    },
    "league_bans": {
      "etf2l": [
        {
          "steam_id": "76561199234205416",
          "alias": "test",
          "expires_at": "2037-12-30T16:00:00-07:00",
          "created_at": "2024-07-07T12:52:19-06:00",
          "reason": "Cheating"
        }
      ],
      "rgl": [
        {
          "steam_id": "76561199234205416",
          "alias": "test",
          "expires_at": "2019-08-31T23:00:00-06:00",
          "created_at": "2019-04-10T22:37:49-06:00",
          "reason": "Using an in-game exploit that messes with hitboxes during playoff match."
        }
      ]
    },
    "logs_count": 0,
    "bot_detector": [
      {
        "list_name": "megacheaterdb",
        "match": {
          "attributes": [
            "cheater"
          ],
          "last_seen": {
            "player_name": "LUCY",
            "time": 1685340466
          },
          "steamid": "76561199234205416",
          "proof": [
            "located at x: 1882.979323141968, y: 610.4778454820851"
          ]
        }
      },
      {
        "list_name": "sleepy-bots",
        "match": {
          "attributes": [
            "cheater"
          ],
          "last_seen": {
            "player_name": "0x0000001a",
            "time": 1681010087
          },
          "steamid": "76561199234205416",
          "proof": [
            "main bot host, see sleepylist for more info",
            "aka LUCY/pokerface/pokerfake"
          ]
        }
      },
      {
        "list_name": "sleepy",
        "match": {
          "attributes": [
            "cheater"
          ],
          "last_seen": {
            "player_name": "winter",
            "time": 1673669662
          },
          "steamid": "76561199234205416",
          "proof": [
            "known as LUCY/pokerface/pokerfake // bot host",
            "**AVOID INTERACTING AT ALL COSTS**",
            "can be commonly seen in DCinside | í¬ì»¤ íŽ˜ì´í¬â·â·â·(122.153): ì»¤ë®¤ì„­ì—ì„œ ì³ë°•í˜€ì„œ ë´‡ ë‘ë ¤ì›Œí•˜ëŠ”ê±° ë¶ˆìŒí•©ë‹ˆë‹¤",
            "https://gall.dcinside.com/board/view/?id=teamfortress2\u0026no=599008",
            "í¬ì»¤ íŽ˜ì´í¬â·â·â·(122.153): í—¨ë¦¬ ì˜¤ëžœë§Œì´ë„¤ ì¹œì‚­ë‹¹í•´ì„œ ë­í•˜ëŠ”ì§€ ëª°ëžëŠ”ë° ì—¬ì „ížˆ ì–´ì„ ëŒë¦¬ë‚˜ë³´ë„¤",
            "ã…‡ã…‡(115.22): ìœ—ìƒˆë‚€ ì¡´ë‚˜ í•µ ë„¤ìž„ë“œìž„",
            "https://gall.dcinside.com/board/view/?id=teamfortress2\u0026no=598195"
          ]
        }
      }
    ],
    "rgl": [
      {
        "division_name": "RGL-Invite",
        "team_leader": "76561199234205416",
        "tag": "TEST",
        "team_name": "test",
        "final_rank": 1,
        "name": "test",
        "is_team_leader": true,
        "joined_at": "2018-06-13T00:05:25.783-06:00",
        "left_at": "0000-12-31T16:26:08-07:33"
      }
    ],
    "friends": [
      {
        "steamid": "76561197961103864",
        "relationship": "friend",
        "friend_since": 1262099119
      },
      {
        "steamid": "76561197962134573",
        "relationship": "friend",
        "friend_since": 1337005818
      }
    ]
  }
]
```

## GET /owned_games

Fetch a list of the users owned games. Note that many users have some or all of this data hidden. If you would
like to forcefully refresh this data, you can append an `&update=true` query value to the request, otherwise
only data older than 2 weeks or data that does not exists yet will be pulled from the steam api.

Supports querying up to 100 steamids at a time, but make sure your timeout is long enough as there is some
pauses in the queries to not hammer valve apis and get rate limited.

Example: https://bd-api.roto.lol/owned_games?steamids=DG_AU

```json
{
    "76561198088775634": [
        {
            "steam_id": "76561198088775634",
            "app_id": 730,
            "name": "Counter-Strike 2",
            "img_icon_url": "8dbc71957312bbd3baea65848b545be9eae2a355",
            "img_logo_url": "",
            "playtime_forever_minutes": 910,
            "playtime_two_weeks": 0,
            "has_community_visible_stats": true,
            "updated_on": "2024-08-01T21:19:24.828453-06:00",
            "created_on": "2024-08-01T21:07:10.549166-06:00"
        },
        {
            "steam_id": "76561198088775634",
            "app_id": 440,
            "name": "Team Fortress 2",
            "img_icon_url": "e3f595a92552da3d664ad00277fad2107345f743",
            "img_logo_url": "",
            "playtime_forever_minutes": 693625,
            "playtime_two_weeks": 1087,
            "has_community_visible_stats": true,
            "updated_on": "2024-08-01T21:19:25.916484-06:00",
            "created_on": "2024-08-01T21:07:12.747306-06:00"
        }
    ]
}

```


## GET /sourcebans

The sourcebans endpoint will return all related data that has been scraped from 3rd party sourcebans sites. There is 
currently close to 100 different sites being scraped.

Entries that have been unbanned on these sites are generally omitted from the data as they can be misleading. Many 
users will blindly assume that *any* sourcebans data equates to a bad actor, when that is usually not the case from 
my experience when crawling these sites. The unbanned users are most often going to be either used as a 
temporary restriction (eg: votekick -> 30min temp ban) or a ban reversal after a false ban / appeal. 

There is no consideration taken into the game being played when the user gets banned, so there is a mix of 
several games in the data such as: TF2, CSGO, GMod, etc. 


Return a map of multiple steam ids: https://bd-api.roto.lol/sourcebans?steamids=76561198976058084
Return a list for a single steam id: https://bd-api.roto.lol/sourcebans/76561198976058084

```json
{
  "76561198976058084": [
    {
      "ban_id": 6723,
      "site_name": "lazypurple",
      "site_id": 61,
      "persona_name": "Shrek",
      "steam_id": "76561198976058084",
      "reason": "griefing; bigotry",
      "duration": 0,
      "permanent": true,
      "created_on": "2023-06-01T17:48:54Z"
    }
  ]
}
```

## GET /bd

Search tracked bot detector lists.

Example: https://bd-api.roto.lol/bd?steamids=76561199176781392

```json
[
    {
        "list_name": "joekiller",
        "match": {
            "attributes": [
                "cheater"
            ],
            "last_seen": {
                "player_name": "Whiâ€te Madnesâ€s",
                "time": 1708860576
            },
            "steamid": "76561199176781392",
            "proof": []
        }
    },
    {
        "list_name": "@trusted",
        "match": {
            "attributes": [
                "cheater"
            ],
            "last_seen": {
                "player_name": "DISCO MOUâ€SEâ€â€â€",
                "time": 1623984811
            },
            "steamid": "76561199176781392",
            "proof": []
        }
    }
]
```

## GET /log/{log_id}

Get a logs.tf match

Examples (truncated): https://bd-api.roto.lol/log/3000

```json
{
    "log_id": 3000,
    "title": "evening-l0108007.log",
    "map": "freight",
    "format": "",
    "duration": 1784,
    "score_red": 3,
    "score_blu": 0,
    "created_on": "2013-01-08T19:46:03-07:00",
    "rounds": [],
    "players": [
        {
            "steam_id": "76561197960460584",
            "team": 3,
            "name": "diaz",
            "classes": null,
            "kills": 21,
            "assists": 12,
            "deaths": 15,
            "damage": 6464,
            "dpm": 217,
            "kad": 2.2,
            "kd": 1.4,
            "damage_taken": 0,
            "dtm": 0,
            "health_packs": 14,
            "backstabs": 13,
            "headshots": 4,
            "airshots": 0,
            "caps": 0,
            "healing_taken": 0
        },
        {
            "steam_id": "76561197989627594",
            "team": 4,
            "name": "Max!",
            "classes": null,
            "kills": 22,
            "assists": 3,
            "deaths": 23,
            "damage": 7319,
            "dpm": 246,
            "kad": 1.1,
            "kd": 1,
            "damage_taken": 0,
            "dtm": 0,
            "health_packs": 41,
            "backstabs": 5,
            "headshots": 2,
            "airshots": 0,
            "caps": 0,
            "healing_taken": 0
        }
    ],
    "medics": [
        {
            "steam_id": "76561197993564443",
            "healing": 6281,
            "healing_per_min": 0,
            "charges_kritz": 0,
            "charges_quickfix": 0,
            "charges_medigun": 18,
            "charges_vacc": 0,
            "drops": 0,
            "avg_time_build": 0,
            "avg_time_use": 0,
            "near_full_death": 0,
            "avg_uber_len": 0,
            "death_after_charge": 0,
            "major_adv_lost": 0,
            "biggest_adv_lost": 0
        },
        {
            "steam_id": "76561198000024718",
            "healing": 6737,
            "healing_per_min": 0,
            "charges_kritz": 0,
            "charges_quickfix": 0,
            "charges_medigun": 8,
            "charges_vacc": 0,
            "drops": 0,
            "avg_time_build": 0,
            "avg_time_use": 0,
            "near_full_death": 0,
            "avg_uber_len": 0,
            "death_after_charge": 0,
            "major_adv_lost": 0,
            "biggest_adv_lost": 0
        }
    ]
}

```

## GET /log/player/{steam_id}

Get a summary of a users logs.tf data.

Example: https://bd-api.roto.lol/log/player/76561197960831093

```json
{
    "logs": 480,
    "kills_avg": 23.02,
    "assists_avg": 9.78,
    "deaths_avg": 16.09,
    "damage_avg": 6338.83,
    "dpm_avg": 262.76,
    "kad_avg": 2.67,
    "kd_avg": 1.89,
    "damage_taken_avg": 782.94,
    "dtm_avg": 37.31,
    "health_packs_avg": 20.41,
    "backstabs_avg": 3.02,
    "headshots_avg": 1.69,
    "airshots_avg": 0,
    "caps_avg": 0.4,
    "healing_taken_avg": 0,
    "kills_sum": 11051,
    "assists_sum": 4696,
    "deaths_sum": 7724,
    "damage_sum": 3042640,
    "damage_taken_sum": 375813,
    "health_packs_sum": 9796,
    "backstabs_sum": 1450,
    "headshots_sum": 813,
    "airshots_sum": 0,
    "caps_sum": 194,
    "healing_taken_sum": 0
}

```

## GET /log/player/{steam_id}/list

Get a high level list of a users logs.tf matches.

Example: https://bd-api.roto.lol/log/player/76561197960831093/list

```json
[
  {
    "log_id": 3,
    "title": "Log 3",
    "map": "",
    "format": "",
    "duration": 1769,
    "score_red": 1,
    "score_blu": 1,
    "created_on": "2012-11-18T14:46:05-07:00"
  },
  {
    "log_id": 13,
    "title": "Log 13",
    "map": "",
    "format": "",
    "duration": 1984,
    "score_red": 5,
    "score_blu": 4,
    "created_on": "2012-11-20T16:15:45-07:00"
  },
  {
    "log_id": 12,
    "title": "Log 12",
    "map": "",
    "format": "",
    "duration": 1769,
    "score_red": 1,
    "score_blu": 1,
    "created_on": "2012-11-20T16:15:22-07:00"
  }
]
```

## GET /serveme

Get a list of current serveme.tf bans.

Example: https://bd-api.roto.lol/serveme

```json
[
    {
        "steam_id": "76561199176100193",
        "name": "bot/cheat dev",
        "reason": "bot/cheat dev",
        "deleted": false,
        "created_on": "2024-07-11T04:27:45.81104-06:00"
    },
    {
        "steam_id": "76561199176117137",
        "name": "bot/cheat dev",
        "reason": "bot/cheat dev",
        "deleted": false,
        "created_on": "2024-07-11T04:27:45.81104-06:00"
    },
    {
        "steam_id": "76561199176183082",
        "name": "bot/cheat dev",
        "reason": "bot/cheat dev",
        "deleted": false,
        "created_on": "2024-07-11T04:27:45.81104-06:00"
    }
]
```

## GET /steamid/{id}

Perform steam id conversions/lookups. Accept any format, including bare vanity name and full profile URLs.

Example: https://bd-api.roto.lol/steamid/76561197961279983

```json
{
    "steam64": "76561197961279983",
    "steam32": 1014255,
    "steam3": "[U:1:1014255]",
    "steam": "STEAM_1:1:507127",
    "profile": "https://steamcommunity.com/profiles/76561197961279983"
}
```

## GET /rgl/player_history

Fetch player team histories for RGL.

Example: https://bd-api.roto.lol/rgl/player_history?steamids=76561197970669109,76561198053621664

```json
{
    "76561197970669109": [
        {
            "division_name": "RGL-Invite",
            "team_leader": "76561197970669109",
            "tag": "FROYO",
            "team_name": "froyotech",
            "final_rank": 1,
            "name": "b4nny",
            "is_team_leader": true,
            "joined_at": "2018-06-13T00:05:25.783-06:00",
            "left_at": "0000-12-31T16:26:08-07:33"
        }
    ],
    "76561198053621664": [
        {
            "division_name": "RGL-Invite",
            "team_leader": "76561197970669109",
            "tag": "FROYO",
            "team_name": "froyotech",
            "final_rank": 1,
            "name": "habib",
            "is_team_leader": false,
            "joined_at": "2018-06-13T03:17:43.753-06:00",
            "left_at": "0000-12-31T16:26:08-07:33"
        }
    ]
}

```

## GET /list/rgl

Return a Bot Detector compatible json result consisting of all known RGL bans.

Example: https://bd-api.roto.lol/list/rgl

```json
{
    "$schema": "https://raw.githubusercontent.com/leighmacdonald/bd-api/master/schemas/playerlist.schema.json",
    "file_info": {
        "authors": [
            "rgl league",
            "bd-api"
        ],
        "description": "All league bans and infractions",
        "title": "RGL.gg Bans",
        "update_url": "http://:8888/list/rgl"
    },
    "players": [
        {
            "attributes": [
                "rgl"
            ],
            "last_seen": {
                "player_name": "maxe0911",
                "time": 1721782795
            },
            "steamid": "76561198157757879",
            "proof": [
                "VAC ban is for a non-TF2 game."
            ]
        },
        {
            "attributes": [
                "rgl"
            ],
            "last_seen": {
                "player_name": "mitty",
                "time": 1721633471
            },
            "steamid": "76561198391550027",
            "proof": [
                "Failure to Submit Demos: 1st Offense"
            ]
        }
    ]
}
```

## GET /list/serveme

Return a Bot Detector compatible json result consisting of all known serveme bans.

Example: https://bd-api.roto.lol/list/serveme

```json
{
  "$schema": "https://raw.githubusercontent.com/leighmacdonald/bd-api/master/schemas/playerlist.schema.json",
  "file_info": {
    "authors": [
      "serveme.tf",
      "bd-api"
    ],
    "description": "All serveme.tf bans",
    "title": "serveme.tf Bans",
    "update_url": "http://:8888/list/serveme"
  },
  "players": [
    {
      "attributes": [
        "serveme"
      ],
      "last_seen": {
        "player_name": "88 street",
        "time": 1722244260
      },
      "steamid": "76561198025169706",
      "proof": [
        "match invader",
        "Permanent Ban"
      ]
    },
    {
      "attributes": [
        "serveme"
      ],
      "last_seen": {
        "player_name": "88 street",
        "time": 1722244260
      },
      "steamid": "76561198065316185",
      "proof": [
        "match invader",
        "Permanent Ban"
      ]
    }
  ]
}
```
## GET /stats

Get the current global stats for the site.

Example: https://bd-api.roto.lol/stats

```json
{
    "bd_list_entries_count": 0,
    "bd_list_count": 0,
    "logs_tf_count": 6392,
    "logs_tf_player_count": 91009,
    "players_count": 383224,
    "sourcebans_sites_count": 88,
    "sourcebans_ban_count": 480527,
    "serveme_ban_count": 697,
    "avatar_count": 383224,
    "name_count": 383224
}
```

## Content Types

If you make API requests with a browser, or otherwise set the `Accept: text/html` header, the JSON output will be encoded
as "prettified" HTML with syntax highlighting of the JSON data returned. All other cases will return standard JSON output.

`curl -H "Accept: text/html" http://localhost:8888/bans\?steamids\=76561197970669109`

```html
<!DOCTYPE html>
<html>
<head> 
        <title>Steam Bans</title>
        <style> body {background-color: #272822;} /* Background */ .bg { color: #f8f8f2; background-color: #272822 }
        ...
        </span></span><span class="line"><span class="cl">    <span class="p">}</span>
        </span></span><span class="line"><span class="cl"><span class="p">]</span></span></span></code></pre>
        </body>
</html>
```
