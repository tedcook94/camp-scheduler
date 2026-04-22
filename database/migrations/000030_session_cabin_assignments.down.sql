-- Reverse migration 000030: restore cabin_id columns and drop session_age_group_cabin_id.
-- Backfill cabin_id from session_age_group_cabins.cabin_id before dropping.

ALTER TABLE counselor_cabin_assignments ADD COLUMN cabin_id uuid;
UPDATE counselor_cabin_assignments a
SET cabin_id = sagc.cabin_id
FROM session_age_group_cabins sagc
WHERE sagc.id = a.session_age_group_cabin_id;
ALTER TABLE counselor_cabin_assignments
    ALTER COLUMN cabin_id SET NOT NULL,
    ADD CONSTRAINT counselor_cabin_assignments_cabin_id_fkey
        FOREIGN KEY (camp_id, cabin_id) REFERENCES cabins(camp_id, id),
    DROP CONSTRAINT counselor_cabin_assignments_session_age_group_cabin_id_fkey,
    DROP COLUMN session_age_group_cabin_id;

ALTER TABLE camper_cabin_assignments ADD COLUMN cabin_id uuid;
UPDATE camper_cabin_assignments a
SET cabin_id = sagc.cabin_id
FROM session_age_group_cabins sagc
WHERE sagc.id = a.session_age_group_cabin_id;
ALTER TABLE camper_cabin_assignments
    ALTER COLUMN cabin_id SET NOT NULL,
    ADD CONSTRAINT camper_cabin_assignments_camp_id_cabin_id_fkey
        FOREIGN KEY (camp_id, cabin_id) REFERENCES cabins(camp_id, id),
    DROP CONSTRAINT camper_cabin_assignments_session_age_group_cabin_id_fkey,
    DROP COLUMN session_age_group_cabin_id;

ALTER TABLE session_age_group_cabins
    DROP CONSTRAINT session_age_group_cabins_camp_id_id_key;
