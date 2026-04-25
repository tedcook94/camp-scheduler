BEGIN;

ALTER TABLE age_groups     ADD COLUMN archived boolean NOT NULL DEFAULT false;
ALTER TABLE cabins         ADD COLUMN archived boolean NOT NULL DEFAULT false;
ALTER TABLE seasons        ADD COLUMN archived boolean NOT NULL DEFAULT false;
ALTER TABLE sessions       ADD COLUMN archived boolean NOT NULL DEFAULT false;
ALTER TABLE campers        ADD COLUMN archived boolean NOT NULL DEFAULT false;
ALTER TABLE activities     ADD COLUMN archived boolean NOT NULL DEFAULT false;
ALTER TABLE time_slots     ADD COLUMN archived boolean NOT NULL DEFAULT false;
ALTER TABLE certifications ADD COLUMN archived boolean NOT NULL DEFAULT false;

ALTER TABLE counselors     ADD COLUMN archived boolean NOT NULL DEFAULT false;
UPDATE counselors SET archived = NOT counselor_enabled;
ALTER TABLE counselors     DROP COLUMN counselor_enabled;

COMMIT;
