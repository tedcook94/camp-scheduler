BEGIN;

-- Add UNIQUE(camp_id, id) constraints to parent tables.
-- These are required for composite foreign keys to reference (camp_id, id).
-- Since id is already a primary key (and therefore unique on its own),
-- (camp_id, id) is always unique, so this is safe on existing data.

ALTER TABLE seasons ADD CONSTRAINT seasons_camp_id_id_key UNIQUE (camp_id, id);
ALTER TABLE sessions ADD CONSTRAINT sessions_camp_id_id_key UNIQUE (camp_id, id);
ALTER TABLE age_groups ADD CONSTRAINT age_groups_camp_id_id_key UNIQUE (camp_id, id);
ALTER TABLE cabins ADD CONSTRAINT cabins_camp_id_id_key UNIQUE (camp_id, id);
ALTER TABLE counselors ADD CONSTRAINT counselors_camp_id_id_key UNIQUE (camp_id, id);
ALTER TABLE session_age_groups ADD CONSTRAINT session_age_groups_camp_id_id_key UNIQUE (camp_id, id);
ALTER TABLE assignment_runs ADD CONSTRAINT assignment_runs_camp_id_id_key UNIQUE (camp_id, id);
ALTER TABLE counselor_cabin_solutions ADD CONSTRAINT counselor_cabin_solutions_camp_id_id_key UNIQUE (camp_id, id);

-- Replace single-column foreign keys with composite (camp_id, foreign_id)
-- foreign keys so the database enforces camp-scoped referential integrity.

-- cabins
ALTER TABLE cabins
    DROP CONSTRAINT cabins_age_group_id_fkey,
    ADD CONSTRAINT cabins_age_group_id_fkey
        FOREIGN KEY (camp_id, age_group_id) REFERENCES age_groups(camp_id, id);

-- sessions
ALTER TABLE sessions
    DROP CONSTRAINT sessions_season_id_fkey,
    ADD CONSTRAINT sessions_season_id_fkey
        FOREIGN KEY (camp_id, season_id) REFERENCES seasons(camp_id, id);

ALTER TABLE sessions
    DROP CONSTRAINT sessions_previous_session_fkey,
    ADD CONSTRAINT sessions_previous_session_fkey
        FOREIGN KEY (camp_id, previous_session) REFERENCES sessions(camp_id, id);

-- session_age_groups
ALTER TABLE session_age_groups
    DROP CONSTRAINT session_age_groups_session_id_fkey,
    ADD CONSTRAINT session_age_groups_session_id_fkey
        FOREIGN KEY (camp_id, session_id) REFERENCES sessions(camp_id, id);

ALTER TABLE session_age_groups
    DROP CONSTRAINT session_age_groups_age_group_id_fkey,
    ADD CONSTRAINT session_age_groups_age_group_id_fkey
        FOREIGN KEY (camp_id, age_group_id) REFERENCES age_groups(camp_id, id);

-- session_age_group_cabins
ALTER TABLE session_age_group_cabins
    DROP CONSTRAINT session_age_group_cabins_session_age_group_id_fkey,
    ADD CONSTRAINT session_age_group_cabins_session_age_group_id_fkey
        FOREIGN KEY (camp_id, session_age_group_id) REFERENCES session_age_groups(camp_id, id);

ALTER TABLE session_age_group_cabins
    DROP CONSTRAINT session_age_group_cabins_cabin_id_fkey,
    ADD CONSTRAINT session_age_group_cabins_cabin_id_fkey
        FOREIGN KEY (camp_id, cabin_id) REFERENCES cabins(camp_id, id);

-- counselor_age_group_preferences
ALTER TABLE counselor_age_group_preferences
    DROP CONSTRAINT counselor_age_group_preferences_counselor_id_fkey,
    ADD CONSTRAINT counselor_age_group_preferences_counselor_id_fkey
        FOREIGN KEY (camp_id, counselor_id) REFERENCES counselors(camp_id, id);

ALTER TABLE counselor_age_group_preferences
    DROP CONSTRAINT counselor_age_group_preferences_session_id_fkey,
    ADD CONSTRAINT counselor_age_group_preferences_session_id_fkey
        FOREIGN KEY (camp_id, session_id) REFERENCES sessions(camp_id, id);

ALTER TABLE counselor_age_group_preferences
    DROP CONSTRAINT counselor_age_group_preferences_age_group_id_fkey,
    ADD CONSTRAINT counselor_age_group_preferences_age_group_id_fkey
        FOREIGN KEY (camp_id, age_group_id) REFERENCES age_groups(camp_id, id);

-- counselor_cocounselor_preferences
ALTER TABLE counselor_cocounselor_preferences
    DROP CONSTRAINT counselor_cocounselor_preferences_counselor_id_fkey,
    ADD CONSTRAINT counselor_cocounselor_preferences_counselor_id_fkey
        FOREIGN KEY (camp_id, counselor_id) REFERENCES counselors(camp_id, id);

ALTER TABLE counselor_cocounselor_preferences
    DROP CONSTRAINT counselor_cocounselor_preferences_session_id_fkey,
    ADD CONSTRAINT counselor_cocounselor_preferences_session_id_fkey
        FOREIGN KEY (camp_id, session_id) REFERENCES sessions(camp_id, id);

ALTER TABLE counselor_cocounselor_preferences
    DROP CONSTRAINT counselor_cocounselor_preferences_preferred_counselor_id_fkey,
    ADD CONSTRAINT counselor_cocounselor_preferences_preferred_counselor_id_fkey
        FOREIGN KEY (camp_id, preferred_counselor_id) REFERENCES counselors(camp_id, id);

-- counselor_session_history
ALTER TABLE counselor_session_history
    DROP CONSTRAINT counselor_session_history_counselor_id_fkey,
    ADD CONSTRAINT counselor_session_history_counselor_id_fkey
        FOREIGN KEY (camp_id, counselor_id) REFERENCES counselors(camp_id, id);

ALTER TABLE counselor_session_history
    DROP CONSTRAINT counselor_session_history_session_id_fkey,
    ADD CONSTRAINT counselor_session_history_session_id_fkey
        FOREIGN KEY (camp_id, session_id) REFERENCES sessions(camp_id, id);

ALTER TABLE counselor_session_history
    DROP CONSTRAINT counselor_session_history_age_group_id_fkey,
    ADD CONSTRAINT counselor_session_history_age_group_id_fkey
        FOREIGN KEY (camp_id, age_group_id) REFERENCES age_groups(camp_id, id);

ALTER TABLE counselor_session_history
    DROP CONSTRAINT counselor_session_history_cabin_id_fkey,
    ADD CONSTRAINT counselor_session_history_cabin_id_fkey
        FOREIGN KEY (camp_id, cabin_id) REFERENCES cabins(camp_id, id);

-- assignment_runs
ALTER TABLE assignment_runs
    DROP CONSTRAINT assignment_runs_session_id_fkey,
    ADD CONSTRAINT assignment_runs_session_id_fkey
        FOREIGN KEY (camp_id, session_id) REFERENCES sessions(camp_id, id);

-- counselor_cabin_solutions
ALTER TABLE counselor_cabin_solutions
    DROP CONSTRAINT counselor_cabin_solutions_assignment_run_id_fkey,
    ADD CONSTRAINT counselor_cabin_solutions_assignment_run_id_fkey
        FOREIGN KEY (camp_id, assignment_run_id) REFERENCES assignment_runs(camp_id, id) ON DELETE CASCADE;

-- counselor_cabin_assignments
ALTER TABLE counselor_cabin_assignments
    DROP CONSTRAINT counselor_cabin_assignments_solution_id_fkey,
    ADD CONSTRAINT counselor_cabin_assignments_solution_id_fkey
        FOREIGN KEY (camp_id, solution_id) REFERENCES counselor_cabin_solutions(camp_id, id) ON DELETE CASCADE;

ALTER TABLE counselor_cabin_assignments
    DROP CONSTRAINT counselor_cabin_assignments_counselor_id_fkey,
    ADD CONSTRAINT counselor_cabin_assignments_counselor_id_fkey
        FOREIGN KEY (camp_id, counselor_id) REFERENCES counselors(camp_id, id);

ALTER TABLE counselor_cabin_assignments
    DROP CONSTRAINT counselor_cabin_assignments_cabin_id_fkey,
    ADD CONSTRAINT counselor_cabin_assignments_cabin_id_fkey
        FOREIGN KEY (camp_id, cabin_id) REFERENCES cabins(camp_id, id);

-- counselor_cabin_explanations
ALTER TABLE counselor_cabin_explanations
    DROP CONSTRAINT counselor_cabin_explanations_solution_id_fkey,
    ADD CONSTRAINT counselor_cabin_explanations_solution_id_fkey
        FOREIGN KEY (camp_id, solution_id) REFERENCES counselor_cabin_solutions(camp_id, id) ON DELETE CASCADE;

ALTER TABLE counselor_cabin_explanations
    DROP CONSTRAINT counselor_cabin_explanations_counselor_id_fkey,
    ADD CONSTRAINT counselor_cabin_explanations_counselor_id_fkey
        FOREIGN KEY (camp_id, counselor_id) REFERENCES counselors(camp_id, id);

-- assignment_runs.selected_solution_id
ALTER TABLE assignment_runs
    DROP CONSTRAINT assignment_runs_selected_solution_id_fkey,
    ADD CONSTRAINT assignment_runs_selected_solution_id_fkey
        FOREIGN KEY (camp_id, selected_solution_id) REFERENCES counselor_cabin_solutions(camp_id, id);

COMMIT;
