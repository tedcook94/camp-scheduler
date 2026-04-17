ALTER TABLE cabins RENAME COLUMN age_group_id TO default_age_group_id;

ALTER TABLE cabins
    DROP CONSTRAINT cabins_age_group_id_fkey,
    ADD CONSTRAINT cabins_default_age_group_id_fkey
        FOREIGN KEY (camp_id, default_age_group_id) REFERENCES age_groups(camp_id, id);
