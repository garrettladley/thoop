-- name: UpsertWorkout :exec
INSERT INTO workouts (id, v1_id, user_id, created_at, updated_at, start, end, timezone_offset, sport_name, score_state, score_json, fetched_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(id) DO UPDATE SET
    v1_id = excluded.v1_id,
    user_id = excluded.user_id,
    created_at = excluded.created_at,
    updated_at = excluded.updated_at,
    start = excluded.start,
    end = excluded.end,
    timezone_offset = excluded.timezone_offset,
    sport_name = excluded.sport_name,
    score_state = excluded.score_state,
    score_json = excluded.score_json,
    fetched_at = CURRENT_TIMESTAMP;

-- name: GetWorkout :one
SELECT * FROM workouts WHERE id = ?;

-- name: GetWorkoutsByDateRange :many
SELECT * FROM workouts
WHERE start >= sqlc.arg(range_start) AND start <= sqlc.arg(range_end)
ORDER BY start DESC
LIMIT sqlc.arg(limit);

-- name: GetWorkoutsByDateRangeCursor :many
SELECT * FROM workouts
WHERE start >= sqlc.arg(range_start) AND start <= sqlc.arg(range_end) AND start < sqlc.arg(cursor)
ORDER BY start DESC
LIMIT sqlc.arg(limit);

-- name: GetWorkoutsByCycleID :many
SELECT * FROM workouts
WHERE workouts.start >= (SELECT cycles.start FROM cycles WHERE cycles.id = sqlc.arg(cycle_id))
  AND workouts.start <= COALESCE((SELECT cycles.end FROM cycles WHERE cycles.id = sqlc.arg(cycle_id)), CURRENT_TIMESTAMP)
ORDER BY workouts.start DESC;

-- name: DeleteWorkout :exec
DELETE FROM workouts WHERE id = ?;
