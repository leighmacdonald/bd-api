# API Reference

All endpoints that support multiple steam ids are limited to a maximum of 100 steam ids per query.

## /bans

Returns the current vac ban states for the requested IDs. 

This returns children with the identical data shape of the [steam web api](https://steamapi.xpaw.me/#ISteamUser/GetPlayerBans).

Example: http://localhost:8888/bans?steamids=76561197970669109,76561197992870439

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

## /friends

Fetch friends lists for the requested steam ids. Depending on the profile visibility states, this will return a empty
list when users set their profiles to private.

This returns a map with children using the identical data shape of the [steam web api](https://steamapi.xpaw.me/#IFriendsListService).

Example: http://localhost:8888/friends?steamids=76561197970669109,76561197992870439

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

## /summary

Fetch player summaries for all requested steam ids. 

Returns a list of player summaries as children. Mirrors data shape of [steam web api](https://steamapi.xpaw.me/#ISteamUser/GetPlayerSummaries).. 

Example: http://localhost:8888/summary?steamids=76561197970669109,76561197992870439

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
## /profile

Profile is a higher level "meta" object that combines the following different data sources into a single object.

- Steam Player Summary
- Steam Vac States
- Steam Friends
- Competitive History RGL/UGC/ETF2L (soon)
- Sourcebans History (soon)
- LogsTF counts (soon)

Example: http://localhost:8888/profile?steamids=76561197970669109,76561197992870439

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

## /sourcebans

The sourcebans endpoint will return all related data that has been scraped from 3rd party sourcebans sites. There is 
currently close to 100 different sites being scraped.

Entries that have been unbanned on these sites are generally omitted from the data as they can be misleading. Many 
users will blindly assume that *any* sourcebans data equates to a bad actor, when that is usually not the case from 
my experience when crawling these sites. The unbanned users are most often going to be either used as a 
temporary restriction (eg: votekick -> 30min temp ban) or a ban reversal after a false ban / appeal. 

There is no consideration taken into the game being played when the user gets banned, so there is a mix of 
several games in the data such as: TF2, CSGO, GMod, etc. 


Return a map of multiple steam ids: http://localhost:8888/sourcebans?steamids=76561198976058084
Return a list for a single steam id: http://localhost:8888/sourcebans/76561198976058084

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
