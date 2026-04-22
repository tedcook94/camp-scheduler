-- Switch counselor_cabin_assignments and camper_cabin_assignments from
-- referencing cabins.id directly to referencing session_age_group_cabins.id
-- so historical assignment rows always carry the session-scoped age-group
-- mapping that was active when the run was generated.

-- Composite unique needed so we can use composite FKs that include camp_id.
ALTER TABLE session_age_group_cabins
    ADD CONSTRAINT session_age_group_cabins_camp_id_id_key UNIQUE (camp_id, id);

-- counselor_cabin_assignments
ALTER TABLE counselor_cabin_assignments
    ADD COLUMN session_age_group_cabin_id uuid;

DO $$
DECLARE
    orphan_count integer;
BEGIN
    UPDATE counselor_cabin_assignments a
    SET session_age_group_cabin_id = sub.sagc_id
    FROM (
        SELECT DISTINCT ON (a.id) a.id AS assignment_id, sagc.id AS sagc_id
        FROM counselor_cabin_assignments a
        JOIN counselor_cabin_solutions ccs ON ccs.id = a.solution_id
        JOIN assignment_runs ar ON ar.id = ccs.assignment_run_id
        JOIN session_age_groups sag ON sag.session_id = ar.session_id
        JOIN session_age_group_cabins sagc
            ON sagc.session_age_group_id = sag.id
            AND sagc.cabin_id = a.cabin_id
        ORDER BY a.id, sagc.id
    ) AS sub
    WHERE a.id = sub.assignment_id;

    SELECT count(*) INTO orphan_count
    FROM counselor_cabin_assignments
    WHERE session_age_group_cabin_id IS NULL;

    IF orphan_count > 0 THEN
        RAISE NOTICE 'Deleting % counselor_cabin_assignments rows that could not be mapped to a session_age_group_cabin', orphan_count;
        DELETE FROM counselor_cabin_assignments WHERE session_age_group_cabin_id IS NULL;
    END IF;
END $$;

ALTER TABLE counselor_cabin_assignments
    ALTER COLUMN session_age_group_cabin_id SET NOT NULL,
    ADD CONSTRAINT counselor_cabin_assignments_session_age_group_cabin_id_fkey
        FOREIGN KEY (camp_id, session_age_group_cabin_id)
        REFERENCES session_age_group_cabins(camp_id, id),
    DROP CONSTRAINT counselor_cabin_assignments_cabin_id_fkey,
    DROP COLUMN cabin_id;

-- camper_cabin_assignments
ALTER TABLE camper_cabin_assignments
    ADD COLUMN session_age_group_cabin_id uuid;

DO $$
DECLARE
    orphan_count integer;
BEGIN
    UPDATE camper_cabin_assignments a
    SET session_age_group_cabin_id = sub.sagc_id
    FROM (
        SELECT DISTINCT ON (a.id) a.id AS assignment_id, sagc.id AS sagc_id
        FROM camper_cabin_assignments a
        JOIN camper_cabin_solutions ccs ON ccs.id = a.solution_id
        JOIN assignment_runs ar ON ar.id = ccs.assignment_run_id
        JOIN session_age_groups sag ON sag.session_id = ar.session_id
        JOIN session_age_group_cabins sagc
            ON sagc.session_age_group_id = sag.id
            AND sagc.cabin_id = a.cabin_id
        ORDER BY a.id, sagc.id
    ) AS sub
    WHERE a.id = sub.assignment_id;

    SELECT count(*) INTO orphan_count
    FROM camper_cabin_assignments
    WHERE session_age_group_cabin_id IS NULL;

    IF orphan_count > 0 THEN
        RAISE NOTICE 'Deleting % camper_cabin_assignments rows that could not be mapped to a session_age_group_cabin', orphan_count;
        DELETE FROM camper_cabin_assignments WHERE session_age_group_cabin_id IS NULL;
    END IF;
END $$;

ALTER TABLE camper_cabin_assignments
    ALTER COLUMN session_age_group_cabin_id SET NOT NULL,
    ADD CONSTRAINT camper_cabin_assignments_session_age_group_cabin_id_fkey
        FOREIGN KEY (camp_id, session_age_group_cabin_id)
        REFERENCES session_age_group_cabins(camp_id, id),
    DROP CONSTRAINT camper_cabin_assignments_camp_id_cabin_id_fkey,
    DROP COLUMN cabin_id;
