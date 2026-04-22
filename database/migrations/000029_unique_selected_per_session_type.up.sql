BEGIN;

-- Add session_id column (initially nullable for backfill)
ALTER TABLE assignment_run_selected_solutions
    ADD COLUMN session_id uuid;

-- Backfill from the join row
UPDATE assignment_run_selected_solutions arss
    SET session_id = ar.session_id
    FROM assignment_runs ar
    WHERE ar.id = arss.run_id AND ar.camp_id = arss.camp_id;

ALTER TABLE assignment_run_selected_solutions
    ALTER COLUMN session_id SET NOT NULL;

ALTER TABLE assignment_run_selected_solutions
    ADD CONSTRAINT arss_camp_session_fkey
    FOREIGN KEY (camp_id, session_id) REFERENCES sessions(camp_id, id);

-- Resolve duplicates: keep the most-recently-created run, drop selections from older ones
WITH ranked AS (
    SELECT arss.run_id,
           arss.camp_id,
           arss.session_id,
           arss.solution_type,
           ROW_NUMBER() OVER (
               PARTITION BY arss.camp_id, arss.session_id, arss.solution_type
               ORDER BY ar.created_at DESC, arss.run_id
           ) AS rn
    FROM assignment_run_selected_solutions arss
    JOIN assignment_runs ar
        ON ar.id = arss.run_id AND ar.camp_id = arss.camp_id
),
losers AS (
    SELECT run_id, camp_id FROM ranked WHERE rn > 1
)
DELETE FROM assignment_run_selected_solutions arss
    USING losers
    WHERE arss.run_id = losers.run_id AND arss.camp_id = losers.camp_id;

-- Revert status on runs that no longer have a selection
UPDATE assignment_runs ar
    SET status = 'completed'
    WHERE ar.status = 'selected'
      AND NOT EXISTS (
          SELECT 1
          FROM assignment_run_selected_solutions arss
          WHERE arss.run_id = ar.id AND arss.camp_id = ar.camp_id
      );

-- Enforce one selection per (camp, session, solution_type)
ALTER TABLE assignment_run_selected_solutions
    ADD CONSTRAINT arss_unique_per_session_type
    UNIQUE (camp_id, session_id, solution_type);

COMMIT;
