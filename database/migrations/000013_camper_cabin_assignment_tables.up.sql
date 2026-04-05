BEGIN;

CREATE TABLE IF NOT EXISTS camper_cabin_solutions (
    id uuid primary key not null default uuid_generate_v4(),
    camp_id uuid not null references camps(id),
    assignment_run_id uuid not null,
    solution_index integer not null,
    score double precision not null,
    score_breakdown jsonb not null default '[]',
    unique (camp_id, id),
    unique (assignment_run_id, solution_index),
    FOREIGN KEY (camp_id, assignment_run_id) REFERENCES assignment_runs(camp_id, id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS camper_cabin_assignments (
    id uuid primary key not null default uuid_generate_v4(),
    camp_id uuid not null references camps(id),
    solution_id uuid not null,
    camper_id uuid not null,
    cabin_id uuid not null,
    unique (solution_id, camper_id),
    FOREIGN KEY (camp_id, solution_id) REFERENCES camper_cabin_solutions(camp_id, id) ON DELETE CASCADE,
    FOREIGN KEY (camp_id, camper_id) REFERENCES campers(camp_id, id),
    FOREIGN KEY (camp_id, cabin_id) REFERENCES cabins(camp_id, id)
);

CREATE TABLE IF NOT EXISTS camper_cabin_explanations (
    id uuid primary key not null default uuid_generate_v4(),
    camp_id uuid not null references camps(id),
    solution_id uuid not null,
    camper_id uuid not null,
    explanation_type text not null,
    constraint_name text,
    message text not null,
    FOREIGN KEY (camp_id, solution_id) REFERENCES camper_cabin_solutions(camp_id, id) ON DELETE CASCADE,
    FOREIGN KEY (camp_id, camper_id) REFERENCES campers(camp_id, id)
);

COMMIT;
