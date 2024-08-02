begin;

DROP TABLE IF EXISTS player_owned_games;
DROP TABLE IF EXISTS steam_game;

DROP INDEX player_updated_on_idx;

ALTER TABLE player ADD COLUMN ugc_updated_on timestamptz not null default created_on;
ALTER TABLE player ADD COLUMN rgl_updated_on timestamptz not null default created_on;
ALTER TABLE player ADD COLUMN etf2l_updated_on timestamptz not null default created_on;
ALTER TABLE player ADD COLUMN logstf_updated_on timestamptz not null default created_on;
ALTER TABLE player RENAME COLUMN updated_on TO steam_updated_on;

commit;
