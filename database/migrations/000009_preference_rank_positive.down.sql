BEGIN;

ALTER TABLE counselor_age_group_preferences
    DROP CONSTRAINT counselor_age_group_preferences_rank_positive;

ALTER TABLE counselor_cocounselor_preferences
    DROP CONSTRAINT counselor_cocounselor_preferences_rank_positive;

COMMIT;
