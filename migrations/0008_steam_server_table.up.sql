begin;

CREATE TABLE IF NOT EXISTS maps
(
    map_id     int PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    map_name   text        not null unique,
    created_on timestamptz not null
);

CREATE TABLE IF NOT EXISTS steam_server
(
    steam_id   bigint      not null primary key,
    addr       inet        not null,
    game_port  int         not null,
    name       text        not null,
    app_id     int         not null,
    game_dir   text        not null,
    version    text        not null,
    region     int         not null,
    secure     bool        not null,
    os         char        not null,
    tags       text[]      not null,
    created_on timestamptz not null,
    updated_on timestamptz not null
);

CREATE TABLE IF NOT EXISTS steam_server_info
(
    steam_id bigint      not null references steam_server (steam_id),
    time     timestamptz not null,
    players  smallint    not null default 0,
    bots     smallint    not null default 0,
    map_id   int         not null references maps (map_id)
);

SELECT create_hypertable('steam_server_info', by_range('time'));

CREATE TABLE IF NOT EXISTS steam_server_counts
(
    time      timestamptz not null,
    valve     int         not null,
    community int         not null,
    linux     int         not null,
    windows   int         not null,
    vac       int         not null,
    sdr       int         not null
);

SELECT create_hypertable('steam_server_counts', by_range('time'));

commit;
