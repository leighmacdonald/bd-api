begin;

CREATE TABLE IF NOT EXISTS steam_game (
    app_id int not null primary key,
    name text not null,
    img_icon_url text not null default '',
    img_logo_url text not null default '',
    created_on timestamptz not null,
    updated_on timestamptz not null
);

CREATE TABLE IF NOT EXISTS player_owned_games (
    steam_id bigint not null references player (steam_id),
    app_id int not null references steam_game (app_id),
    playtime_forever_minutes int not null default 0, -- minutes
    playtime_two_weeks int not null default 0,
    has_community_visible_stats bool not null default false,
    created_on timestamptz not null,
    updated_on timestamptz not null
);

CREATE UNIQUE INDEX player_owned_games_uidx ON player_owned_games (steam_id, app_id);

ALTER TABLE player DROP COLUMN ugc_updated_on;
ALTER TABLE player DROP COLUMN rgl_updated_on;
ALTER TABLE player DROP COLUMN etf2l_updated_on;
ALTER TABLE player DROP COLUMN logstf_updated_on;
ALTER TABLE player RENAME COLUMN steam_updated_on TO updated_on;

CREATE INDEX player_updated_on_idx ON player USING brin (updated_on);

commit;
