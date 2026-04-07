BEGIN;

CREATE TABLE IF NOT EXISTS activity_solutions (
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

CREATE TABLE IF NOT EXISTS activity_assignments (
    id uuid primary key not null default uuid_generate_v4(),
    camp_id uuid not null references camps(id),
    solution_id uuid not null,
    counselor_id uuid not null,
    session_activity_id uuid not null,
    FOREIGN KEY (camp_id, solution_id) REFERENCES activity_solutions(camp_id, id) ON DELETE CASCADE,
    FOREIGN KEY (camp_id, counselor_id) REFERENCES counselors(camp_id, id),
    FOREIGN KEY (camp_id, session_activity_id) REFERENCES session_activities(camp_id, id)
);

CREATE TABLE IF NOT EXISTS activity_explanations (
    id uuid primary key not null default uuid_generate_v4(),
    camp_id uuid not null references camps(id),
    solution_id uuid not null,
    counselor_id uuid not null,
    explanation_type text not null,
    constraint_name text,
    message text not null,
    FOREIGN KEY (camp_id, solution_id) REFERENCES activity_solutions(camp_id, id) ON DELETE CASCADE,
    FOREIGN KEY (camp_id, counselor_id) REFERENCES counselors(camp_id, id)
);

-- Widen the solution_type check to include the new activity_schedule type.
-- The inline CHECK from migration 000014 has a Postgres-generated name,
-- so we look it up dynamically before dropping it.
DO $$
DECLARE
    _con text;
BEGIN
    SELECT conname INTO _con
    FROM pg_constraint
    WHERE conrelid = 'assignment_run_selected_solutions'::regclass
      AND contype = 'c'
      AND pg_get_constraintdef(oid) LIKE '%solution_type%';
    IF _con IS NOT NULL THEN
        EXECUTE format('ALTER TABLE assignment_run_selected_solutions DROP CONSTRAINT %I', _con);
    END IF;
END
$$;

ALTER TABLE assignment_run_selected_solutions
    ADD CONSTRAINT arss_solution_type_check
    CHECK (solution_type IN ('counselor_cabin', 'camper_cabin', 'activity_schedule'));

COMMIT;
