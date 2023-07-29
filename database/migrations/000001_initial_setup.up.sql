BEGIN;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS camps (
    id uuid primary key not null default uuid_generate_v4(),
    camp_name text unique not null,
    camp_location text,
    camp_enabled boolean not null default true
);

CREATE TABLE IF NOT EXISTS age_groups (
    id uuid primary key not null default uuid_generate_v4(),
    camp_id uuid not null references camps(id),
    age_group_name text not null,
    unique (camp_id, age_group_name)
);

CREATE TABLE IF NOT EXISTS cabins (
    id uuid primary key not null default uuid_generate_v4(),
    camp_id uuid not null references camps(id),
    age_group_id uuid not null references age_groups(id),
    cabin_name text not null,
    unique (camp_id, cabin_name)
);

CREATE TABLE IF NOT EXISTS seasons (
    id uuid primary key not null default uuid_generate_v4(),
    camp_id uuid not null references camps(id),
    season_name text not null,
    unique (camp_id, season_name)
);

CREATE TABLE IF NOT EXISTS sessions (
    id uuid primary key not null default uuid_generate_v4(),
    camp_id uuid not null references camps(id),
    season_id uuid not null references seasons(id),
    session_name text not null,
    previous_session uuid references sessions(id),
    unique (season_id, session_name)
);

CREATE TABLE IF NOT EXISTS session_age_groups (
    id uuid primary key not null default uuid_generate_v4(),
    camp_id uuid not null references camps(id),
    session_id uuid not null references sessions(id),
    age_group_id uuid not null references age_groups(id),
    group_size integer,
    unique (session_id, age_group_id)
);

CREATE TABLE IF NOT EXISTS session_age_group_cabins (
    id uuid primary key not null default uuid_generate_v4(),
    camp_id uuid not null references camps(id),
    session_age_group_id uuid not null references session_age_groups(id),
    cabin_id uuid not null references cabins(id),
    group_size integer,
    required_counselors integer,
    unique (session_age_group_id, cabin_id)
);

CREATE TABLE IF NOT EXISTS counselors (
    id uuid primary key not null default uuid_generate_v4(),
    camp_id uuid not null references camps(id),
    counselor_name text not null,
    junior_counselor boolean not null default false,
    counselor_enabled boolean not null default true
);

COMMIT;
