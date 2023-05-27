begin;

create type league AS ENUM ('rgl', 'ugc', 'etf2l', 'esea');

create table if not exists player
(
    steam_id                 bigint
        constraint player_pk primary key,
    ip_addr                  text default '' not null,
    communityvisibilitystate int  default 0  not null,
    profilestate             int             not null,
    persona_name             text            not null,
    vanity                   text            not null,
    avatarhash               text            not null,
    personastate             int             not null,
    realname                 text            not null,
    timecreated              int             not null,
    loccountrycode           text            not null,
    locstatecode             text            not null,
    loccityid                int             not null,
    community_banned         boolean         not null,
    vac_banned               boolean         not null,
    game_bans                int             not null,
    economy_ban              int             not null,
    logstf_count             int             not null,
    ugc_updated_at           timestamp       not null,
    rgl_updated_at           timestamp       not null,
    etf2l_updated_at         timestamp       not null,
    logstf_updated_at        timestamp       not null,
    steam_updated_on         timestamp       not null,
    created_on               timestamp       not null
);

create table if not exists league
(
    league_id   int
        constraint league_pk primary key,
    league_name text unique,
    created_on  timestamp default now() not null
);

create table if not exists team
(
    season_id    bigint primary key,
    steam_id     bigint references player,
    league      league not null,
    division     text not null,
    division_int int  not null,
    format       text not null,
    name         text not null
);

commit;
