-- name: ListSessionTimeSlots :many
SELECT id, camp_id, session_id, time_slot_id, sort_order
FROM session_time_slots
WHERE session_id = $1 AND camp_id = $2
ORDER BY sort_order, time_slot_id;

-- name: GetSessionTimeSlot :one
SELECT id, camp_id, session_id, time_slot_id, sort_order
FROM session_time_slots
WHERE id = $1 AND camp_id = $2 AND session_id = $3;

-- name: CreateSessionTimeSlot :one
INSERT INTO session_time_slots (camp_id, session_id, time_slot_id, sort_order)
VALUES ($1, $2, $3, $4)
RETURNING id, camp_id, session_id, time_slot_id, sort_order;

-- name: UpdateSessionTimeSlot :one
UPDATE session_time_slots
SET time_slot_id = $4,
    sort_order = $5
WHERE id = $1 AND camp_id = $2 AND session_id = $3
RETURNING id, camp_id, session_id, time_slot_id, sort_order;

-- name: DeleteSessionTimeSlot :execrows
DELETE FROM session_time_slots
WHERE id = $1 AND camp_id = $2 AND session_id = $3;

-- name: UpdateSessionTimeSlotSortOrder :exec
UPDATE session_time_slots
SET sort_order = $4
WHERE id = $1 AND camp_id = $2 AND session_id = $3;
