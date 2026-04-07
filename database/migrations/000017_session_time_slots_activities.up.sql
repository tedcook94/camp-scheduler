BEGIN;

CREATE TABLE IF NOT EXISTS session_time_slots (
    id uuid primary key not null default uuid_generate_v4(),
    camp_id uuid not null references camps(id),
    session_id uuid not null,
    time_slot_id uuid not null,
    sort_order integer not null default 0,
    unique (session_id, time_slot_id),
    unique (camp_id, id),
    FOREIGN KEY (camp_id, session_id) REFERENCES sessions(camp_id, id),
    FOREIGN KEY (camp_id, time_slot_id) REFERENCES time_slots(camp_id, id)
);

CREATE TABLE IF NOT EXISTS session_activities (
    id uuid primary key not null default uuid_generate_v4(),
    camp_id uuid not null references camps(id),
    session_time_slot_id uuid not null,
    activity_id uuid not null,
    capacity integer not null default 1,
    required_counselors integer not null default 1,
    unique (session_time_slot_id, activity_id),
    unique (camp_id, id),
    FOREIGN KEY (camp_id, session_time_slot_id) REFERENCES session_time_slots(camp_id, id) ON DELETE CASCADE,
    FOREIGN KEY (camp_id, activity_id) REFERENCES activities(camp_id, id),
    CHECK (capacity > 0),
    CHECK (required_counselors > 0),
    CHECK (required_counselors <= capacity)
);

COMMIT;
