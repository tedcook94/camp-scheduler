BEGIN;

ALTER TABLE activity_explanations
    ADD COLUMN session_activity_id uuid;

ALTER TABLE activity_explanations
    ADD CONSTRAINT activity_explanations_camp_id_session_activity_id_fkey
    FOREIGN KEY (camp_id, session_activity_id)
    REFERENCES session_activities(camp_id, id)
    ON DELETE CASCADE;

COMMIT;
