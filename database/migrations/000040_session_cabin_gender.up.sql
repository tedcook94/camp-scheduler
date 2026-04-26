ALTER TABLE session_age_group_cabins
    ADD COLUMN gender text NOT NULL DEFAULT 'female'
        CHECK (gender IN ('male', 'female'));

UPDATE session_age_group_cabins sagc
SET gender = c.gender
FROM cabins c
WHERE c.id = sagc.cabin_id;

ALTER TABLE session_age_group_cabins ALTER COLUMN gender DROP DEFAULT;
