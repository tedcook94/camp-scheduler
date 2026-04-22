BEGIN;

DROP INDEX IF EXISTS session_counselors_counselor_id_idx;
DROP INDEX IF EXISTS session_counselors_session_id_idx;
DROP TABLE IF EXISTS session_counselors;

COMMIT;
