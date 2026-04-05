BEGIN;

CREATE TABLE IF NOT EXISTS assignment_runs (
    id uuid primary key not null default uuid_generate_v4(),
    camp_id uuid not null references camps(id),
    session_id uuid not null references sessions(id),
    run_type text not null,
    status text not null default 'completed',
    created_at timestamptz not null default now()
);

CREATE TABLE IF NOT EXISTS counselor_cabin_solutions (
    id uuid primary key not null default uuid_generate_v4(),
    camp_id uuid not null references camps(id),
    assignment_run_id uuid not null references assignment_runs(id) on delete cascade,
    solution_index integer not null,
    score double precision not null,
    score_breakdown jsonb not null default '[]',
    unique (assignment_run_id, solution_index)
);

CREATE TABLE IF NOT EXISTS counselor_cabin_assignments (
    id uuid primary key not null default uuid_generate_v4(),
    camp_id uuid not null references camps(id),
    solution_id uuid not null references counselor_cabin_solutions(id) on delete cascade,
    counselor_id uuid not null references counselors(id),
    cabin_id uuid not null references cabins(id),
    unique (solution_id, counselor_id)
);

CREATE TABLE IF NOT EXISTS counselor_cabin_explanations (
    id uuid primary key not null default uuid_generate_v4(),
    camp_id uuid not null references camps(id),
    solution_id uuid not null references counselor_cabin_solutions(id) on delete cascade,
    counselor_id uuid not null references counselors(id),
    explanation_type text not null,
    constraint_name text,
    message text not null
);

COMMIT;
