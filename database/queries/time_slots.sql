-- name: ListTimeSlots :many
SELECT id, camp_id, time_slot_name, archived
FROM time_slots
WHERE camp_id = $1 AND archived = false
ORDER BY time_slot_name;

-- name: ListAllTimeSlots :many
SELECT id, camp_id, time_slot_name, archived
FROM time_slots
WHERE camp_id = $1
ORDER BY time_slot_name;

-- name: ListArchivedTimeSlots :many
SELECT id, camp_id, time_slot_name, archived
FROM time_slots
WHERE camp_id = $1 AND archived = true
ORDER BY time_slot_name;

-- name: GetTimeSlot :one
SELECT id, camp_id, time_slot_name, archived
FROM time_slots
WHERE id = $1 AND camp_id = $2;

-- name: CreateTimeSlot :one
INSERT INTO time_slots (camp_id, time_slot_name)
VALUES ($1, $2)
RETURNING id, camp_id, time_slot_name, archived;

-- name: UpdateTimeSlot :one
UPDATE time_slots
SET time_slot_name = $3
WHERE id = $1 AND camp_id = $2
RETURNING id, camp_id, time_slot_name, archived;

-- name: DeleteTimeSlot :execrows
DELETE FROM time_slots
WHERE id = $1 AND camp_id = $2;
