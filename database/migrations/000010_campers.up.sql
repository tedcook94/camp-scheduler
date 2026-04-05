BEGIN;

CREATE TABLE IF NOT EXISTS campers (
    id uuid primary key not null default uuid_generate_v4(),
    camp_id uuid not null references camps(id),
    camper_name text not null,
    unique (camp_id, id)
);

COMMIT;
