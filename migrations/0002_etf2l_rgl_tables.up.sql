begin;

create table etf2l_player
(
    steam_id   bigint primary key references player (steam_id) on delete cascade,
    id         bigint      not null,
    name       text        not null default '',
    country    text        not null default '',
    created_on timestamptz not null,
    updated_on timestamptz not null
);

create table etf2l_bans
(
    ban_id     serial primary key,
    steam_id   bigint      not null
        references etf2l_player (steam_id) on delete cascade,
    start_date timestamptz not null,
    end_date   timestamptz not null,
    reason     text        not null,
    created_on timestamptz not null,
    updated_on timestamptz not null
);

create table etf2l_team
(
    team_id    int primary key,
    name       text        not null,
    country    text        not null,
    -- type       text        not null,
    tag        text        not null default '',
    created_on timestamptz not null,
    updated_on timestamptz not null
);

create table etf2l_team_player
(
    team_id    int primary key references etf2l_team (team_id) on delete cascade,
    steam_id   bigint      not null
        references etf2l_player (steam_id) on delete cascade,
    created_on timestamptz not null,
    updated_on timestamptz not null
);

create unique index etf2l_team_player_uidx ON etf2l_team_player (team_id, steam_id);

create table etf2l_competition
(
    competition_id int primary key,
    team_id        int         not null
        references etf2l_team (team_id) on delete cascade,
    category       text        not null,
    competition    text        not null,
    created_on     timestamptz not null,
    updated_on     timestamptz not null
);

-- create table etf2l_competition_table
-- (
--     competition_id int primary key,
--     team_id        int  not null references etf2l_team (team_id) on delete cascade,
--     division_name  text not null,
--     division_tier  int  not null,
--     country        text not null default '',
--     name           text not null,
--     maps_played    int  not null default 0,
--     maps_won       int  not null default 0,
--     gc_won         int  not null default 0,
--     gc_lost        int  not null default 0,
--     maps_lost      int  not null default 0,
--     penalty_points int  not null default 0,
--     score          int  not null default 0,
--     ach            int  not null default 0,
--     byes           int  not null default 0,
--     seeded_points  int  not null default 0
-- );
--
-- create unique index etf2l_team_player_uidx ON etf2l_competition_table (competition_id, team_id);
--
-- create table etf2l_competition_result
-- (
--     competition_result_id int primary key,
--     competition_id int not null,
--     clan1_team_id        int  not null references etf2l_team (team_id) on delete cascade,
--     clan1_drop bool not null,
--     clan1_country text not null default '',
--     clan1_name text not null default '',
--     division_name  text not null,
--     division_tier  int  not null
--
-- );

create table rgl_team
(
    team_id       int primary key,
    name          text        not null,
    tag           text        not null default '',
    division_name text        not null,
    division_id   int         not null,
    final_rank    int         not null default 0,
    season_id     int         not null default 0,
    created_on    timestamptz not null,
    updated_on    timestamptz not null
);

create unique index if not exists rgl_team_uidx ON rgl_team (team_id, season_id);

create table rgl_player
(
    steam_id        bigint primary key references player (steam_id),
    avatar          text        not null default '',
    name            text        not null default '',
    updated_at      timestamptz not null,
    is_verified     bool        not null,
    is_banned       bool        not null,
    is_on_probation bool        not null,
    created_on      timestamptz not null,
    updated_on      timestamptz not null
);

create table rgl_team_player
(
    team_id    int primary key references rgl_team (team_id),
    name       text        not null,
    steam_id   bigint      not null references rgl_player (steam_id),
    is_leader  bool        not null default false,
    joined_at  timestamptz not null,
    left_at    timestamptz,
    created_on timestamptz not null,
    updated_on timestamptz not null
);

create unique index if not exists rgl_team_player_uidx ON rgl_team_player (team_id, steam_id);

create table rgl_ban
(
    steam_id   bigint primary key references player (steam_id),
    name       text        not null,
    expires_at timestamptz not null,
    created_at timestamptz not null,
    reason     text        not null,
    created_on timestamptz not null,
    updated_on timestamptz not null
);

create unique index if not exists rgl_ban_uidx ON rgl_ban (steam_id, created_at);

commit;
