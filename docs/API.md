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
- Sourcebans History (soon)
- LogsTF counts (soon)

Example: https://bd-api.roto.lol/profile?steamids=76561197970669109,76561197992870439

```json
[
    {
        "summary": {
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
        },
        "ban_state": {
            "SteamId": "76561197970669109",
            "CommunityBanned": false,
            "VACBanned": false,
            "NumberOfVACBans": 0,
            "DaysSinceLastBan": 0,
            "NumberOfGameBans": 0,
            "EconomyBan": "none"
        },
        "seasons": null,
        "friends": [
            {
                "steamid": "76561197960265749",
                "relationship": "friend",
                "friend_since": 1449274040
            },
            {
                "steamid": "76561197960269040",
                "relationship": "friend",
                "friend_since": 1449272149
            }
        ],
        "logs_count": 0
    },
    {
        "summary": {
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
        "ban_state": {
            "SteamId": "76561197992870439",
            "CommunityBanned": false,
            "VACBanned": false,
            "NumberOfVACBans": 0,
            "DaysSinceLastBan": 0,
            "NumberOfGameBans": 0,
            "EconomyBan": "none"
        },
        "seasons": null,
        "friends": [],
        "logs_count": 0
    }
]
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
