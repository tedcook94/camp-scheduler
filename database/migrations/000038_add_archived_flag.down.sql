BEGIN;

ALTER TABLE counselors     ADD COLUMN counselor_enabled boolean NOT NULL DEFAULT true;
UPDATE counselors SET counselor_enabled = NOT archived;
ALTER TABLE counselors     DROP COLUMN archived;

ALTER TABLE certifications DROP COLUMN archived;
ALTER TABLE time_slots     DROP COLUMN archived;
ALTER TABLE activities     DROP COLUMN archived;
ALTER TABLE campers        DROP COLUMN archived;
ALTER TABLE sessions       DROP COLUMN archived;
ALTER TABLE seasons        DROP COLUMN archived;
ALTER TABLE cabins         DROP COLUMN archived;
ALTER TABLE age_groups     DROP COLUMN archived;

COMMIT;
