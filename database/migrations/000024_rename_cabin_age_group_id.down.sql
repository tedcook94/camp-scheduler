ALTER TABLE cabins RENAME COLUMN default_age_group_id TO age_group_id;

ALTER TABLE cabins
    DROP CONSTRAINT cabins_default_age_group_id_fkey,
    ADD CONSTRAINT cabins_age_group_id_fkey
        FOREIGN KEY (camp_id, age_group_id) REFERENCES age_groups(camp_id, id);
