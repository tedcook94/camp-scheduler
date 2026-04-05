BEGIN;

CREATE TABLE IF NOT EXISTS camper_session_enrollments (
    id uuid primary key not null default uuid_generate_v4(),
    camp_id uuid not null references camps(id),
    camper_id uuid not null,
    session_id uuid not null,
    session_age_group_id uuid not null,
    unique (camp_id, id),
    unique (camper_id, session_age_group_id),
    unique (camper_id, session_id),
    FOREIGN KEY (camp_id, camper_id) REFERENCES campers(camp_id, id),
    FOREIGN KEY (camp_id, session_id) REFERENCES sessions(camp_id, id),
    FOREIGN KEY (camp_id, session_age_group_id) REFERENCES session_age_groups(camp_id, id)
);

CREATE OR REPLACE FUNCTION validate_enrollment_session_age_group()
RETURNS trigger AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM session_age_groups
        WHERE camp_id = NEW.camp_id
          AND session_id = NEW.session_id
          AND id = NEW.session_age_group_id
    ) THEN
        RAISE EXCEPTION
            'session_age_group_id % does not belong to session_id % for camp_id %',
            NEW.session_age_group_id,
            NEW.session_id,
            NEW.camp_id;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER enrollment_session_age_group_consistency
BEFORE INSERT OR UPDATE OF camp_id, session_id, session_age_group_id
ON camper_session_enrollments
FOR EACH ROW
EXECUTE FUNCTION validate_enrollment_session_age_group();

COMMIT;
