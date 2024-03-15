begin;

create table if not exists bd_list
(
    bd_list_id   serial primary key,
    bd_list_name text      not null unique CHECK ( length(bd_list_name) > 0 ),
    url          text      not null unique CHECK ( length(url) > 0 ),
    game         text      not null default 'tf2',
    trust_weight int       not null default 1 CHECK ( trust_weight >= 0 AND trust_weight <= 10 ),
    deleted      bool      not null default false,
    created_on   timestamp not null,
    updated_on   timestamp not null
);


create table if not exists bd_list_entries
(
    bd_list_entry_id bigserial primary key,
    bd_list_id       int       not null
        constraint bd_list_fk
            references bd_list (bd_list_id) on delete cascade,
    steam_id         bigint references player CHECK ( steam_id > 76561197960265728 ),
    attribute        text[]    not null CHECK ( array_length(attribute, 1) > 0 ),
    proof            text[]    not null default '{}'::text[],
    last_seen        timestamp not null,
    last_name        text      not null default '',
    deleted          boolean   not null default false,
    created_on       timestamp not null,
    updated_on       timestamp not null
);

create unique index if not exists bd_list_entries_uidx ON bd_list_entries (bd_list_id, steam_id);

commit;
