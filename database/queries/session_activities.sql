-- name: ListSessionActivities :many
SELECT sa.id, sa.camp_id, sa.session_time_slot_id, sa.activity_id, sa.capacity, sa.required_counselors
FROM session_activities sa
JOIN session_time_slots sts ON sts.id = sa.session_time_slot_id
WHERE sts.session_id = $1 AND sa.camp_id = $2
ORDER BY sts.sort_order, sa.activity_id;

-- name: ListSessionActivitiesByTimeSlot :many
SELECT id, camp_id, session_time_slot_id, activity_id, capacity, required_counselors
FROM session_activities
WHERE session_time_slot_id = $1 AND camp_id = $2
ORDER BY activity_id;

-- name: GetSessionActivity :one
SELECT sa.id, sa.camp_id, sa.session_time_slot_id, sa.activity_id, sa.capacity, sa.required_counselors
FROM session_activities sa
JOIN session_time_slots sts ON sts.id = sa.session_time_slot_id
WHERE sa.id = $1 AND sa.camp_id = $2 AND sts.session_id = $3;

-- name: CreateSessionActivity :one
INSERT INTO session_activities (camp_id, session_time_slot_id, activity_id, capacity, required_counselors)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, camp_id, session_time_slot_id, activity_id, capacity, required_counselors;

-- name: UpdateSessionActivity :one
UPDATE session_activities sa
SET activity_id = $4,
    capacity = $5,
    required_counselors = $6
FROM session_time_slots sts
WHERE sa.id = $1 AND sa.camp_id = $2
  AND sts.id = sa.session_time_slot_id AND sts.session_id = $3
RETURNING sa.id, sa.camp_id, sa.session_time_slot_id, sa.activity_id, sa.capacity, sa.required_counselors;

-- name: DeleteSessionActivity :execrows
DELETE FROM session_activities sa
USING session_time_slots sts
WHERE sa.id = $1 AND sa.camp_id = $2
  AND sts.id = sa.session_time_slot_id AND sts.session_id = $3;

-- name: ListSessionActivitiesWithDetails :many
SELECT
    sa.id AS session_activity_id,
    sa.capacity,
    sa.required_counselors,
    a.id AS activity_id,
    a.activity_name,
    sts.id AS session_time_slot_id,
    sts.time_slot_id,
    ts.time_slot_name,
    sts.sort_order
FROM session_activities sa
JOIN activities a ON a.id = sa.activity_id
JOIN session_time_slots sts ON sts.id = sa.session_time_slot_id
JOIN time_slots ts ON ts.id = sts.time_slot_id
WHERE sts.session_id = $1 AND sa.camp_id = $2
ORDER BY sts.sort_order, a.activity_name;

-- name: DeleteSessionActivitiesByTimeSlot :execrows
DELETE FROM session_activities
WHERE session_time_slot_id = $1 AND camp_id = $2;

-- name: ListActivityCertificationsBySession :many
SELECT
    sa.id AS session_activity_id,
    ac.certification_id
FROM session_activities sa
JOIN session_time_slots sts ON sts.id = sa.session_time_slot_id
JOIN activity_certifications ac ON ac.activity_id = sa.activity_id AND ac.camp_id = sa.camp_id
WHERE sts.session_id = $1 AND sa.camp_id = $2
ORDER BY sa.id, ac.certification_id;
