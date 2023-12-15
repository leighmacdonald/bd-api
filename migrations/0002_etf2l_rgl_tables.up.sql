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
    id    int primary key,
    name       text        not null,
    country    text        not null,
    created_on timestamptz not null,
    updated_on timestamptz not null
);

create table etf2l_team_player
(
    team_id  int primary key references etf2l_team (id) on delete cascade,
    steam_id bigint not null
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
    division_name  text        not null,
    division_tier  int         not null,
    created_on     timestamptz not null,
    updated_on     timestamptz not null
);

commit;
