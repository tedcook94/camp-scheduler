BEGIN;

CREATE TABLE IF NOT EXISTS counselor_activity_preferences (
    id uuid primary key not null default uuid_generate_v4(),
    camp_id uuid not null references camps(id),
    counselor_id uuid not null,
    session_id uuid not null,
    activity_id uuid not null,
    rank integer not null,
    unique (counselor_id, session_id, activity_id),
    FOREIGN KEY (camp_id, counselor_id) REFERENCES counselors(camp_id, id),
    FOREIGN KEY (camp_id, session_id) REFERENCES sessions(camp_id, id),
    FOREIGN KEY (camp_id, activity_id) REFERENCES activities(camp_id, id),
    CHECK (rank > 0)
);

COMMIT;
