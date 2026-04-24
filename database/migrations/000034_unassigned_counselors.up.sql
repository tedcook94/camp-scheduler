BEGIN;

-- Tracks counselors who could not be placed in any cabin by a counselor
-- cabin solution. Cabin runs that placed every counselor have no rows here.
CREATE TABLE IF NOT EXISTS counselor_cabin_unassigned (
    id uuid primary key not null default uuid_generate_v4(),
    camp_id uuid not null references camps(id),
    solution_id uuid not null,
    counselor_id uuid not null,
    unique (camp_id, id),
    unique (solution_id, counselor_id),
    FOREIGN KEY (camp_id, solution_id) REFERENCES counselor_cabin_solutions(camp_id, id) ON DELETE CASCADE,
    FOREIGN KEY (camp_id, counselor_id) REFERENCES counselors(camp_id, id) ON DELETE CASCADE
);

-- Tracks counselors who were not placed in every time slot they had at
-- least one eligible activity slot for. Each parent row carries the list
-- of missing time slots in counselor_activity_unassigned_slots.
CREATE TABLE IF NOT EXISTS counselor_activity_unassigned (
    id uuid primary key not null default uuid_generate_v4(),
    camp_id uuid not null references camps(id),
    solution_id uuid not null,
    counselor_id uuid not null,
    unique (camp_id, id),
    unique (solution_id, counselor_id),
    FOREIGN KEY (camp_id, solution_id) REFERENCES activity_solutions(camp_id, id) ON DELETE CASCADE,
    FOREIGN KEY (camp_id, counselor_id) REFERENCES counselors(camp_id, id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS counselor_activity_unassigned_slots (
    id uuid primary key not null default uuid_generate_v4(),
    camp_id uuid not null references camps(id),
    unassigned_id uuid not null,
    session_time_slot_id uuid not null,
    unique (unassigned_id, session_time_slot_id),
    FOREIGN KEY (camp_id, unassigned_id) REFERENCES counselor_activity_unassigned(camp_id, id) ON DELETE CASCADE,
    FOREIGN KEY (camp_id, session_time_slot_id) REFERENCES session_time_slots(camp_id, id) ON DELETE CASCADE
);

COMMIT;
