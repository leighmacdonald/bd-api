begin;

create table if not exists bd_list
(
    bd_list_id   int primary key,
    bd_list_name text not null unique,
    url       text not null unique,
    game      text not null default 'tf2',
    trust_weight int not null default 1 CHECK ( trust_weight >= 0 AND trust_weight <= 10 ),
    deleted bool not null default false,
    created_on timestamp not null,
    updated_on timestamp not null
);

create table if not exists bd_list_entries
(
    bd_list_entry_id bigint primary key,
    bd_list_id       int not null
        constraint bd_list_fk
            references bd_list (bd_list_id) on delete cascade,
    steam_id      bigint references player CHECK ( steam_id > 76561197960265728 ),
    attribute text not null,
    last_seen timestamp not null,
    last_name text not null default '',
    deleted boolean not null default false,
    created_on timestamp not null,
    updated_on timestamp not null
);

commit;
