BEGIN;

CREATE TABLE IF NOT EXISTS certifications (
    id uuid primary key not null default uuid_generate_v4(),
    camp_id uuid not null references camps(id),
    certification_name text not null,
    unique (camp_id, certification_name),
    unique (camp_id, id)
);

CREATE TABLE IF NOT EXISTS activities (
    id uuid primary key not null default uuid_generate_v4(),
    camp_id uuid not null references camps(id),
    activity_name text not null,
    unique (camp_id, activity_name),
    unique (camp_id, id)
);

CREATE TABLE IF NOT EXISTS activity_certifications (
    id uuid primary key not null default uuid_generate_v4(),
    camp_id uuid not null references camps(id),
    activity_id uuid not null,
    certification_id uuid not null,
    unique (activity_id, certification_id),
    FOREIGN KEY (camp_id, activity_id) REFERENCES activities(camp_id, id) ON DELETE CASCADE,
    FOREIGN KEY (camp_id, certification_id) REFERENCES certifications(camp_id, id)
);

CREATE TABLE IF NOT EXISTS time_slots (
    id uuid primary key not null default uuid_generate_v4(),
    camp_id uuid not null references camps(id),
    time_slot_name text not null,
    unique (camp_id, time_slot_name),
    unique (camp_id, id)
);

COMMIT;
