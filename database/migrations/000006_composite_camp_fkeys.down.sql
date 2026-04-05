BEGIN;

-- Revert composite foreign keys back to single-column foreign keys
-- and drop the UNIQUE(camp_id, id) constraints.

-- assignment_runs.selected_solution_id
ALTER TABLE assignment_runs
    DROP CONSTRAINT assignment_runs_selected_solution_id_fkey,
    ADD CONSTRAINT assignment_runs_selected_solution_id_fkey
        FOREIGN KEY (selected_solution_id) REFERENCES counselor_cabin_solutions(id);

-- counselor_cabin_explanations
ALTER TABLE counselor_cabin_explanations
    DROP CONSTRAINT counselor_cabin_explanations_counselor_id_fkey,
    ADD CONSTRAINT counselor_cabin_explanations_counselor_id_fkey
        FOREIGN KEY (counselor_id) REFERENCES counselors(id);

ALTER TABLE counselor_cabin_explanations
    DROP CONSTRAINT counselor_cabin_explanations_solution_id_fkey,
    ADD CONSTRAINT counselor_cabin_explanations_solution_id_fkey
        FOREIGN KEY (solution_id) REFERENCES counselor_cabin_solutions(id) ON DELETE CASCADE;

-- counselor_cabin_assignments
ALTER TABLE counselor_cabin_assignments
    DROP CONSTRAINT counselor_cabin_assignments_cabin_id_fkey,
    ADD CONSTRAINT counselor_cabin_assignments_cabin_id_fkey
        FOREIGN KEY (cabin_id) REFERENCES cabins(id);

ALTER TABLE counselor_cabin_assignments
    DROP CONSTRAINT counselor_cabin_assignments_counselor_id_fkey,
    ADD CONSTRAINT counselor_cabin_assignments_counselor_id_fkey
        FOREIGN KEY (counselor_id) REFERENCES counselors(id);

ALTER TABLE counselor_cabin_assignments
    DROP CONSTRAINT counselor_cabin_assignments_solution_id_fkey,
    ADD CONSTRAINT counselor_cabin_assignments_solution_id_fkey
        FOREIGN KEY (solution_id) REFERENCES counselor_cabin_solutions(id) ON DELETE CASCADE;

-- counselor_cabin_solutions
ALTER TABLE counselor_cabin_solutions
    DROP CONSTRAINT counselor_cabin_solutions_assignment_run_id_fkey,
    ADD CONSTRAINT counselor_cabin_solutions_assignment_run_id_fkey
        FOREIGN KEY (assignment_run_id) REFERENCES assignment_runs(id) ON DELETE CASCADE;

-- assignment_runs
ALTER TABLE assignment_runs
    DROP CONSTRAINT assignment_runs_session_id_fkey,
    ADD CONSTRAINT assignment_runs_session_id_fkey
        FOREIGN KEY (session_id) REFERENCES sessions(id);

-- counselor_session_history
ALTER TABLE counselor_session_history
    DROP CONSTRAINT counselor_session_history_cabin_id_fkey,
    ADD CONSTRAINT counselor_session_history_cabin_id_fkey
        FOREIGN KEY (cabin_id) REFERENCES cabins(id);

ALTER TABLE counselor_session_history
    DROP CONSTRAINT counselor_session_history_age_group_id_fkey,
    ADD CONSTRAINT counselor_session_history_age_group_id_fkey
        FOREIGN KEY (age_group_id) REFERENCES age_groups(id);

ALTER TABLE counselor_session_history
    DROP CONSTRAINT counselor_session_history_session_id_fkey,
    ADD CONSTRAINT counselor_session_history_session_id_fkey
        FOREIGN KEY (session_id) REFERENCES sessions(id);

ALTER TABLE counselor_session_history
    DROP CONSTRAINT counselor_session_history_counselor_id_fkey,
    ADD CONSTRAINT counselor_session_history_counselor_id_fkey
        FOREIGN KEY (counselor_id) REFERENCES counselors(id);

-- counselor_cocounselor_preferences
ALTER TABLE counselor_cocounselor_preferences
    DROP CONSTRAINT counselor_cocounselor_preferences_preferred_counselor_id_fkey,
    ADD CONSTRAINT counselor_cocounselor_preferences_preferred_counselor_id_fkey
        FOREIGN KEY (preferred_counselor_id) REFERENCES counselors(id);

ALTER TABLE counselor_cocounselor_preferences
    DROP CONSTRAINT counselor_cocounselor_preferences_session_id_fkey,
    ADD CONSTRAINT counselor_cocounselor_preferences_session_id_fkey
        FOREIGN KEY (session_id) REFERENCES sessions(id);

ALTER TABLE counselor_cocounselor_preferences
    DROP CONSTRAINT counselor_cocounselor_preferences_counselor_id_fkey,
    ADD CONSTRAINT counselor_cocounselor_preferences_counselor_id_fkey
        FOREIGN KEY (counselor_id) REFERENCES counselors(id);

-- counselor_age_group_preferences
ALTER TABLE counselor_age_group_preferences
    DROP CONSTRAINT counselor_age_group_preferences_age_group_id_fkey,
    ADD CONSTRAINT counselor_age_group_preferences_age_group_id_fkey
        FOREIGN KEY (age_group_id) REFERENCES age_groups(id);

ALTER TABLE counselor_age_group_preferences
    DROP CONSTRAINT counselor_age_group_preferences_session_id_fkey,
    ADD CONSTRAINT counselor_age_group_preferences_session_id_fkey
        FOREIGN KEY (session_id) REFERENCES sessions(id);

ALTER TABLE counselor_age_group_preferences
    DROP CONSTRAINT counselor_age_group_preferences_counselor_id_fkey,
    ADD CONSTRAINT counselor_age_group_preferences_counselor_id_fkey
        FOREIGN KEY (counselor_id) REFERENCES counselors(id);

-- session_age_group_cabins
ALTER TABLE session_age_group_cabins
    DROP CONSTRAINT session_age_group_cabins_cabin_id_fkey,
    ADD CONSTRAINT session_age_group_cabins_cabin_id_fkey
        FOREIGN KEY (cabin_id) REFERENCES cabins(id);

ALTER TABLE session_age_group_cabins
    DROP CONSTRAINT session_age_group_cabins_session_age_group_id_fkey,
    ADD CONSTRAINT session_age_group_cabins_session_age_group_id_fkey
        FOREIGN KEY (session_age_group_id) REFERENCES session_age_groups(id);

-- session_age_groups
ALTER TABLE session_age_groups
    DROP CONSTRAINT session_age_groups_age_group_id_fkey,
    ADD CONSTRAINT session_age_groups_age_group_id_fkey
        FOREIGN KEY (age_group_id) REFERENCES age_groups(id);

ALTER TABLE session_age_groups
    DROP CONSTRAINT session_age_groups_session_id_fkey,
    ADD CONSTRAINT session_age_groups_session_id_fkey
        FOREIGN KEY (session_id) REFERENCES sessions(id);

-- sessions
ALTER TABLE sessions
    DROP CONSTRAINT sessions_previous_session_fkey,
    ADD CONSTRAINT sessions_previous_session_fkey
        FOREIGN KEY (previous_session) REFERENCES sessions(id);

ALTER TABLE sessions
    DROP CONSTRAINT sessions_season_id_fkey,
    ADD CONSTRAINT sessions_season_id_fkey
        FOREIGN KEY (season_id) REFERENCES seasons(id);

-- cabins
ALTER TABLE cabins
    DROP CONSTRAINT cabins_age_group_id_fkey,
    ADD CONSTRAINT cabins_age_group_id_fkey
        FOREIGN KEY (age_group_id) REFERENCES age_groups(id);

-- Drop UNIQUE(camp_id, id) constraints
ALTER TABLE counselor_cabin_solutions DROP CONSTRAINT counselor_cabin_solutions_camp_id_id_key;
ALTER TABLE assignment_runs DROP CONSTRAINT assignment_runs_camp_id_id_key;
ALTER TABLE session_age_groups DROP CONSTRAINT session_age_groups_camp_id_id_key;
ALTER TABLE counselors DROP CONSTRAINT counselors_camp_id_id_key;
ALTER TABLE cabins DROP CONSTRAINT cabins_camp_id_id_key;
ALTER TABLE age_groups DROP CONSTRAINT age_groups_camp_id_id_key;
ALTER TABLE sessions DROP CONSTRAINT sessions_camp_id_id_key;
ALTER TABLE seasons DROP CONSTRAINT seasons_camp_id_id_key;

COMMIT;
