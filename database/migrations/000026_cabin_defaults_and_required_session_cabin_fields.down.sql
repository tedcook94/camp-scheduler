ALTER TABLE session_age_group_cabins
    ALTER COLUMN group_size DROP NOT NULL,
    ALTER COLUMN required_counselors DROP NOT NULL;

ALTER TABLE cabins
    DROP COLUMN default_group_size,
    DROP COLUMN default_required_counselors;
