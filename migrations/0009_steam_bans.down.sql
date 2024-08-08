begin;

DROP TABLE IF EXISTS player_bans;

ALTER TABLE player ADD COLUMN IF NOT EXISTS community_banned bool not null;
ALTER TABLE player ADD COLUMN IF NOT EXISTS vac_banned bool not null;
ALTER TABLE player ADD COLUMN IF NOT EXISTS game_bans int not null;
ALTER TABLE player ADD COLUMN IF NOT EXISTS economy_banned int not null;

commit;
