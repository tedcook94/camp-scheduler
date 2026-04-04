BEGIN;

CREATE TABLE IF NOT EXISTS counselor_session_history (
    id uuid primary key not null default uuid_generate_v4(),
    camp_id uuid not null references camps(id),
    counselor_id uuid not null references counselors(id),
    session_id uuid not null references sessions(id),
    age_group_id uuid not null references age_groups(id),
    cabin_id uuid references cabins(id),
    unique (counselor_id, session_id)
);

COMMIT;
