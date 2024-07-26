begin;

CREATE TABLE IF NOT EXISTS rgl_season
(
    season_id           int primary key,
    season_name         text        not null,
    maps                text[]      not null,
    format_name         text        not null default '',
    region_name         text        not null default '',
    participating_teams int[]       not null,
    created_on          timestamptz not null
);

CREATE TABLE IF NOT EXISTS rgl_team
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

CREATE TABLE IF NOT EXISTS rgl_team_member
(
    team_id int not null references rgl_team (team_id),
    steam_id bigint not null references player (steam_id),
    name text not null,
    is_team_leader bool not null,
    joined_at timestamptz not null,
    left_at timestamptz
);

CREATE UNIQUE INDEX rgl_team_member_uidx ON rgl_team_member (team_id, steam_id);

create table if not exists rgl_ban
(
    steam_id   bigint primary key references player (steam_id),
    alias      text        not null,
    expires_at timestamptz not null,
    created_at timestamptz not null,
    reason     text        not null
);

CREATE UNIQUE INDEX rgl_ban_uidx ON rgl_ban (steam_id, created_at);

-- create table if not exists rgl_match
-- (
--     match_id int primary key
-- );

commit;
