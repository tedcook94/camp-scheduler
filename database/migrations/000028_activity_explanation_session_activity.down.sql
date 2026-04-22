BEGIN;

ALTER TABLE activity_explanations
    DROP CONSTRAINT IF EXISTS activity_explanations_camp_id_session_activity_id_fkey;

ALTER TABLE activity_explanations
    DROP COLUMN IF EXISTS session_activity_id;

COMMIT;
