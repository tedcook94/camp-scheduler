-- Add cabin defaults with placeholder 1, then drop the DEFAULT so future
-- inserts must specify the values explicitly.
ALTER TABLE cabins
    ADD COLUMN default_group_size integer NOT NULL DEFAULT 1,
    ADD COLUMN default_required_counselors integer NOT NULL DEFAULT 1;

ALTER TABLE cabins
    ALTER COLUMN default_group_size DROP DEFAULT,
    ALTER COLUMN default_required_counselors DROP DEFAULT;

-- Backfill any null session_age_group_cabins values, then enforce NOT NULL.
UPDATE session_age_group_cabins
SET group_size = COALESCE(group_size, 1),
    required_counselors = COALESCE(required_counselors, 1);

ALTER TABLE session_age_group_cabins
    ALTER COLUMN group_size SET NOT NULL,
    ALTER COLUMN required_counselors SET NOT NULL;
