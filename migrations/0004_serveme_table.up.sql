begin;

create table if not exists serveme
(
    steam_id   bigint primary key references player (steam_id),
    name       text        not null default '',
    reason     text        not null default '',
    deleted    boolean     not null default false,
    created_on timestamptz not null,
    updated_on  timestamptz not null
);

commit;
