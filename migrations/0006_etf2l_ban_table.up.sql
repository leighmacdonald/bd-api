begin;

CREATE TABLE IF NOT EXISTS etf2l_ban (
    steam_id bigint not null references player (steam_id),
    alias text not null,
    created_at timestamptz not null,
    expires_at timestamptz not null,
    reason text not null
);

CREATE UNIQUE INDEX etf2l_ban_idx ON etf2l_ban (steam_id, created_at);

commit;
