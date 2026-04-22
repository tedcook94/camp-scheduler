ALTER TABLE counselor_cabin_explanations ADD COLUMN rank INTEGER;
ALTER TABLE camper_cabin_explanations ADD COLUMN rank INTEGER;
ALTER TABLE activity_explanations ADD COLUMN rank INTEGER;

WITH ranked AS (
  SELECT id, ROW_NUMBER() OVER (
    PARTITION BY solution_id, counselor_id, explanation_type
    ORDER BY id
  ) AS rn
  FROM counselor_cabin_explanations
)
UPDATE counselor_cabin_explanations e
SET rank = ranked.rn
FROM ranked
WHERE ranked.id = e.id;

WITH ranked AS (
  SELECT id, ROW_NUMBER() OVER (
    PARTITION BY solution_id, camper_id, explanation_type
    ORDER BY id
  ) AS rn
  FROM camper_cabin_explanations
)
UPDATE camper_cabin_explanations e
SET rank = ranked.rn
FROM ranked
WHERE ranked.id = e.id;

WITH ranked AS (
  SELECT id, ROW_NUMBER() OVER (
    PARTITION BY solution_id, counselor_id, explanation_type
    ORDER BY id
  ) AS rn
  FROM activity_explanations
)
UPDATE activity_explanations e
SET rank = ranked.rn
FROM ranked
WHERE ranked.id = e.id;
