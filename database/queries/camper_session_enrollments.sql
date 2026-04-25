-- name: ListSessionEnrollments :many
SELECT
    e.id,
    e.camp_id,
    e.camper_id,
    e.session_age_group_id,
    c.first_name AS camper_first_name,
    c.last_name AS camper_last_name,
    btrim(c.first_name || ' ' || c.last_name)::text AS camper_name,
    c.gender,
    c.archived AS camper_archived,
    sag.session_id,
    sag.age_group_id
FROM camper_session_enrollments e
JOIN campers c ON c.id = e.camper_id AND c.camp_id = e.camp_id
JOIN session_age_groups sag ON sag.id = e.session_age_group_id AND sag.camp_id = e.camp_id
WHERE sag.session_id = $1 AND e.camp_id = $2
ORDER BY c.last_name, c.first_name;

-- name: GetSessionEnrollment :one
SELECT
    e.id,
    e.camp_id,
    e.camper_id,
    e.session_age_group_id,
    c.first_name AS camper_first_name,
    c.last_name AS camper_last_name,
    btrim(c.first_name || ' ' || c.last_name)::text AS camper_name,
    sag.session_id,
    sag.age_group_id
FROM camper_session_enrollments e
JOIN campers c ON c.id = e.camper_id AND c.camp_id = e.camp_id
JOIN session_age_groups sag ON sag.id = e.session_age_group_id AND sag.camp_id = e.camp_id
WHERE e.id = $1 AND e.camp_id = $2 AND e.session_id = $3;

-- name: CreateSessionEnrollment :one
INSERT INTO camper_session_enrollments (camp_id, camper_id, session_id, session_age_group_id)
VALUES ($1, $2, $3, $4)
RETURNING id, camp_id, camper_id, session_id, session_age_group_id;

-- name: DeleteSessionEnrollment :execrows
DELETE FROM camper_session_enrollments
WHERE id = $1 AND camp_id = $2 AND session_id = $3;

-- name: ListEnrollmentsByCamper :many
SELECT
    e.id,
    e.camp_id,
    e.camper_id,
    e.session_age_group_id,
    c.first_name AS camper_first_name,
    c.last_name AS camper_last_name,
    btrim(c.first_name || ' ' || c.last_name)::text AS camper_name,
    sag.session_id,
    sag.age_group_id
FROM camper_session_enrollments e
JOIN campers c ON c.id = e.camper_id AND c.camp_id = e.camp_id
JOIN session_age_groups sag ON sag.id = e.session_age_group_id AND sag.camp_id = e.camp_id
JOIN sessions s ON s.id = sag.session_id AND s.camp_id = e.camp_id
JOIN seasons se ON se.id = s.season_id AND se.camp_id = e.camp_id
WHERE e.camper_id = $1 AND e.camp_id = $2
ORDER BY se.start_date DESC, s.session_name ASC;

-- name: ListEnrollmentsBySessionAgeGroup :many
SELECT
    e.id,
    e.camp_id,
    e.camper_id,
    e.session_age_group_id,
    c.first_name AS camper_first_name,
    c.last_name AS camper_last_name,
    btrim(c.first_name || ' ' || c.last_name)::text AS camper_name,
    sag.session_id,
    sag.age_group_id
FROM camper_session_enrollments e
JOIN campers c ON c.id = e.camper_id AND c.camp_id = e.camp_id
JOIN session_age_groups sag ON sag.id = e.session_age_group_id AND sag.camp_id = e.camp_id
WHERE e.session_age_group_id = $1 AND e.camp_id = $2
ORDER BY c.last_name, c.first_name;
