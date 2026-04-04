BEGIN;

CREATE TABLE IF NOT EXISTS counselor_age_group_preferences (
    id uuid primary key not null default uuid_generate_v4(),
    camp_id uuid not null references camps(id),
    counselor_id uuid not null references counselors(id),
    session_id uuid not null references sessions(id),
    age_group_id uuid not null references age_groups(id),
    rank integer not null,
    unique (counselor_id, session_id, age_group_id),
    unique (counselor_id, session_id, rank)
);

CREATE TABLE IF NOT EXISTS counselor_cocounselor_preferences (
    id uuid primary key not null default uuid_generate_v4(),
    camp_id uuid not null references camps(id),
    counselor_id uuid not null references counselors(id),
    session_id uuid not null references sessions(id),
    preferred_counselor_id uuid not null references counselors(id),
    rank integer not null,
    check (counselor_id != preferred_counselor_id),
    unique (counselor_id, session_id, preferred_counselor_id),
    unique (counselor_id, session_id, rank)
);

COMMIT;
