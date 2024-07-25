begin;

create table rgl_season
(
    season_id           int primary key,
    season_name         text        not null,
    maps                text[]      not null,
    format_name         text        not null default '',
    region_name         text        not null default '',
    participating_teams int[]       not null,
    created_on          timestamptz not null
);

create table if not exists rgl_team
(
    team_id       int primary key,
    season_id     int         not null references rgl_season (season_id),
    division_id   int         not null,
    division_name text        not null,
    team_leader   bigint      not null references player (steam_id),
    tag           text        not null default '',
    team_name     text        not null default '',
    final_rank    int         not null default 0,
--     team_status text not null,
--     team_ready bool,
    created_at    timestamptz not null,
    updated_at    timestamptz not null
);

-- create table if not exists rgl_season_team
-- (
--     season_id int not null references rgl_season (season_id),
--     team_id int not null references rgl_team (team_id)
-- );

create table if not exists rgl_player
(
    steam_id        bigint primary key references player (steam_id),
    player_name     text        not null,
    verified        bool        not null,
    updated_at      timestamptz not null,
    is_banned       bool        not null,
    is_on_probation bool        not null,
    created_on      timestamptz not null
);

create table if not exists rgl_bans
(
    steam_id   bigint primary key references player (steam_id),
    alias      text        not null,
    expires_at timestamptz not null,
    created_on timestamptz not null,
    reason     text        not null
);



create table if not exists rgl_match
(
    match_id int primary key

);

commit;
