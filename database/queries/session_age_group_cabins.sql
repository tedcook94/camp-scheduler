-- name: ListSessionCabins :many
SELECT
    c.id,
    c.cabin_name,
    c.age_group_id,
    sagc.required_counselors
FROM session_age_groups sag
JOIN session_age_group_cabins sagc ON sagc.session_age_group_id = sag.id
JOIN cabins c ON c.id = sagc.cabin_id
WHERE sag.session_id = $1 AND sag.camp_id = $2
ORDER BY c.cabin_name;
