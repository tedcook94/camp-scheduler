ALTER TABLE assignment_runs
ADD COLUMN selected_solution_id uuid REFERENCES counselor_cabin_solutions(id);
