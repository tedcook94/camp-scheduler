ALTER TABLE assignment_runs
    DROP CONSTRAINT assignment_runs_selected_solution_id_fkey,
    ADD CONSTRAINT assignment_runs_selected_solution_id_fkey
        FOREIGN KEY (camp_id, selected_solution_id) REFERENCES counselor_cabin_solutions(camp_id, id) ON DELETE SET NULL;
