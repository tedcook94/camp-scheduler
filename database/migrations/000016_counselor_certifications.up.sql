BEGIN;

CREATE TABLE IF NOT EXISTS counselor_certifications (
    id uuid primary key not null default uuid_generate_v4(),
    camp_id uuid not null references camps(id),
    counselor_id uuid not null,
    certification_id uuid not null,
    unique (counselor_id, certification_id),
    FOREIGN KEY (camp_id, counselor_id) REFERENCES counselors(camp_id, id) ON DELETE CASCADE,
    FOREIGN KEY (camp_id, certification_id) REFERENCES certifications(camp_id, id)
);

COMMIT;
