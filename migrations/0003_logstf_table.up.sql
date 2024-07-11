begin;

create table if not exists logstf
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

create table if not exists logstf_player
(
    log_id        bigint not null references logstf (log_id) ON DELETE CASCADE,
    steam_id      bigint not null references player (steam_id),
    team          int    not null,
    name          text   not null,
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

create unique index logstf_player_uidx ON logstf_player (log_id, steam_id);

create table if not exists logstf_round (
    log_id        bigint not null references logstf (log_id) ON DELETE CASCADE,
    round int not null,
    length bigint not null,
    score_blu int not null,
    score_red int not null,
    kills_blu int not null,
    kills_red int not null,
    ubers_blu int not null,
    ubers_red int not null,
    damage_blu int not null,
    damage_red int not null,
    midfight int not null
);

create unique index logstf_round_uidx ON logstf_round(log_id, round);

create table if not exists logstf_player_class
(
    log_id   bigint not null references logstf (log_id) ON DELETE CASCADE,
    steam_id bigint not null references player (steam_id),
    player_class int not null,
    played bigint not null,
    kills int not null,
    assists int not null,
    deaths int not null,
    damage bigint not null
);

create unique index logstf_player_class_uidx ON logstf_player_class (log_id, steam_id, player_class);

create table if not exists logstf_player_class_weapon
(
    log_id   bigint not null references logstf (log_id) ON DELETE CASCADE,
    steam_id bigint not null references player (steam_id),
    weapon text not null,
    kills int not null,
    damage int not null,
    accuracy int not null
);

create unique index logstf_player_class_weapon_uidx ON logstf_player_class_weapon (log_id, steam_id, weapon);

create table if not exists logstf_medic
(
    log_id             bigint not null references logstf (log_id) ON DELETE CASCADE,
    steam_id           bigint not null references player (steam_id),
    healing            int    not null default 0,
    charges_kritz      int    not null default 0,
    charges_quickfix   int    not null default 0,
    charges_medigun    int    not null default 0,
    charges_vacc       int    not null default 0,
    avg_time_build     bigint    not null default 0,
    avg_time_use       bigint    not null default 0,
    near_full_death    int    not null default 0,
    avg_uber_len       float  not null default 0,
    death_after_charge int    not null default 0,
    major_adv_lost     int    not null default 0,
    biggest_adv_lost   bigint    not null default 0
);

create unique index if not exists logstf_medic_uidx ON logstf_medic (log_id, steam_id);

commit;
