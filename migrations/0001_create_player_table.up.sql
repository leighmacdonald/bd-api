begin;

create table if not exists player
(
    steam_id                 bigint
        constraint player_pk primary key,
    communityvisibilitystate int default 0 not null,
    profilestate             int           not null,
    persona_name             text          not null,
    vanity                   text          not null,
    avatarhash               text          not null,
    personastate             int           not null,
    realname                 text          not null,
    timecreated              int           not null,
    loccountrycode           text          not null,
    locstatecode             text          not null,
    loccityid                int           not null,
    community_banned         boolean       not null,
    vac_banned               boolean       not null,
    game_bans                int           not null,
    economy_ban              int           not null,
    logstf_count             int           not null,
    ugc_updated_on           timestamp     not null,
    rgl_updated_on           timestamp     not null,
    etf2l_updated_on         timestamp     not null,
    logstf_updated_on        timestamp     not null,
    steam_updated_on         timestamp     not null,
    created_on               timestamp     not null
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
    league_id    int  not null
        constraint team_league_fk
            references league (league_id) on delete cascade,
    division     text not null,
    division_int int  not null,
    format       text not null,
    name         text not null
);

create table if not exists sb_site
(
    sb_site_id bigint
        constraint sb_site_pk
            primary key,
    name       text not null
);

create index if not exists sb_site_uidx ON sb_site (name);

create table if not exists sb_ban
(
    sb_ban_id  bigserial
        constraint sb_ban_pk
            primary key,
    sb_site_id int       not null
        constraint ban_site_fk
            references sb_site (sb_site_id) on delete cascade,
    steam_id   bigint    not null
        constraint ban_steam_fk
            references player (steam_id) on delete cascade,
    reason     text      not null,
    created_on timestamp not null,
    duration   bigint    not null,
    permanent  boolean   not null
);

create index if not exists sb_ban_uidx ON sb_ban (sb_site_id, steam_id, created_on);

commit;
