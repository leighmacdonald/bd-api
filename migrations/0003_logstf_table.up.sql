begin;

create table if not exists logtf_log
(
    log_id     bigint primary key,
    title      text        not null default '',
    map        text        not null default '',
    format     text        not null default '',
    views      int         not null default 0,
    duration   bigint      not null default 0,
    score_red  int         not null default 0,
    score_blu  int         not null default 0,
    created_on timestamptz not null
);

create table if not exists logtf_log_player
(
    log_id        bigint not null references logtf_log (log_id) ON DELETE CASCADE,
    steam_id      bigint not null references player (steam_id),
    team          int    not null,
    name          text   not null,
    classes       int[]  not null,
    kills         int    not null default 0,
    assists       int    not null default 0,
    deaths        int    not null default 0,
    damage        int    not null default 0,
    dpm           int    not null default 0,
    kad           float  not null default 0,
    kd            float  not null default 0,
    dt            int    not null default 0,
    dtm           int    not null default 0,
    hp            int    not null default 0,
    bs            int    not null default 0,
    hs            int    not null default 0,
    caps          int    not null default 0,
    healing_taken int    not null default 0
);

create table if not exists logtf_log_medic
(
    log_id             bigint not null references logtf_log (log_id) ON DELETE CASCADE,
    steam_id           bigint not null references player (steam_id),
    healing            int    not null default 0,
    charges_kritz      int    not null default 0,
    charges_quickfix   int    not null default 0,
    charges_medigun    int    not null default 0,
    charges_vacc       int    not null default 0,
    avg_time_build     int    not null default 0,
    avt_time_use       int    not null default 0,
    near_full_death    int    not null default 0,
    avg_uber_len       float  not null default 0,
    death_after_charge int    not null default 0,
    major_adv_lost     int    not null default 0,
    biggest_adv_lost   int    not null default 0
);

create unique index if not exists logtf_log_player_uidx ON logtf_log_player (log_id, steam_id);
create unique index if not exists logtf_log_medic_uidx ON logtf_log_medic (log_id, steam_id);

commit;
