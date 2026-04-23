-- Add gender to cabins, counselors, and campers. Default existing rows to
-- 'female', then drop the DEFAULT so future inserts must specify explicitly.
ALTER TABLE cabins
    ADD COLUMN gender text NOT NULL DEFAULT 'female'
        CHECK (gender IN ('male', 'female'));
ALTER TABLE cabins
    ALTER COLUMN gender DROP DEFAULT;

ALTER TABLE counselors
    ADD COLUMN gender text NOT NULL DEFAULT 'female'
        CHECK (gender IN ('male', 'female'));
ALTER TABLE counselors
    ALTER COLUMN gender DROP DEFAULT;

ALTER TABLE campers
    ADD COLUMN gender text NOT NULL DEFAULT 'female'
        CHECK (gender IN ('male', 'female'));
ALTER TABLE campers
    ALTER COLUMN gender DROP DEFAULT;
