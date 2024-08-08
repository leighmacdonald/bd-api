begin;

ALTER TABLE player
    DROP COLUMN IF EXISTS community_banned;
ALTER TABLE player
    DROP COLUMN IF EXISTS vac_banned;
ALTER TABLE player
    DROP COLUMN IF EXISTS game_bans;
ALTER TABLE player
    DROP COLUMN IF EXISTS economy_banned;


CREATE TYPE economy_ban_types AS ENUM ('none', 'probation', 'banned');

CREATE TABLE IF NOT EXISTS player_bans
(
    steam_id            bigint primary key references player (steam_id),
    community_banned    bool              not null default false,
    vac_banned          bool              not null default false,
    number_of_vac_bans  int               not null default 0,
    number_of_game_bans int               not null default 0,
    economy_ban         economy_ban_types not null default 'none',
    created_on          timestamptz       not null,
    updated_on          timestamptz       not null
);

commit;
