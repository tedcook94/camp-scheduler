BEGIN;

ALTER TABLE counselor_age_group_preferences
    ADD CONSTRAINT counselor_age_group_preferences_rank_positive CHECK (rank > 0);

ALTER TABLE counselor_cocounselor_preferences
    ADD CONSTRAINT counselor_cocounselor_preferences_rank_positive CHECK (rank > 0);

COMMIT;
