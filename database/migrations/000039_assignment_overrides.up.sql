BEGIN;

-- Pre-solver overrides. An override pins a specific assignment that the
-- solver must honor as a forced placement before searching for the rest.
-- Validation against hard constraints is performed at save time and
-- re-checked when a run is triggered.

CREATE TABLE IF NOT EXISTS counselor_cabin_overrides (
    id uuid primary key not null default uuid_generate_v4(),
    camp_id uuid not null references camps(id),
    session_id uuid not null,
    counselor_id uuid not null,
    session_age_group_cabin_id uuid not null,
    created_at timestamptz not null default now(),
    unique (camp_id, id),
    unique (session_id, counselor_id),
    FOREIGN KEY (camp_id, session_id) REFERENCES sessions(camp_id, id) ON DELETE CASCADE,
    FOREIGN KEY (camp_id, counselor_id) REFERENCES counselors(camp_id, id) ON DELETE CASCADE,
    FOREIGN KEY (camp_id, session_age_group_cabin_id) REFERENCES session_age_group_cabins(camp_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS counselor_cabin_overrides_session_id_idx ON counselor_cabin_overrides (session_id);

CREATE TABLE IF NOT EXISTS camper_cabin_overrides (
    id uuid primary key not null default uuid_generate_v4(),
    camp_id uuid not null references camps(id),
    session_id uuid not null,
    camper_id uuid not null,
    session_age_group_cabin_id uuid not null,
    created_at timestamptz not null default now(),
    unique (camp_id, id),
    unique (session_id, camper_id),
    FOREIGN KEY (camp_id, session_id) REFERENCES sessions(camp_id, id) ON DELETE CASCADE,
    FOREIGN KEY (camp_id, camper_id) REFERENCES campers(camp_id, id) ON DELETE CASCADE,
    FOREIGN KEY (camp_id, session_age_group_cabin_id) REFERENCES session_age_group_cabins(camp_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS camper_cabin_overrides_session_id_idx ON camper_cabin_overrides (session_id);

-- A counselor may pin multiple activities (one per time slot). Time-slot
-- conflicts are rejected at save time, not by a DB constraint.
CREATE TABLE IF NOT EXISTS counselor_activity_overrides (
    id uuid primary key not null default uuid_generate_v4(),
    camp_id uuid not null references camps(id),
    session_id uuid not null,
    counselor_id uuid not null,
    session_activity_id uuid not null,
    created_at timestamptz not null default now(),
    unique (camp_id, id),
    unique (session_id, counselor_id, session_activity_id),
    FOREIGN KEY (camp_id, session_id) REFERENCES sessions(camp_id, id) ON DELETE CASCADE,
    FOREIGN KEY (camp_id, counselor_id) REFERENCES counselors(camp_id, id) ON DELETE CASCADE,
    FOREIGN KEY (camp_id, session_activity_id) REFERENCES session_activities(camp_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS counselor_activity_overrides_session_id_idx ON counselor_activity_overrides (session_id);

COMMIT;
