-- name: UpsertSleep :exec
INSERT INTO sleeps (id, cycle_id, v1_id, user_id, created_at, updated_at, start, end, timezone_offset, nap, score_state, score_json, fetched_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(id) DO UPDATE SET
    cycle_id = excluded.cycle_id,
    v1_id = excluded.v1_id,
    user_id = excluded.user_id,
    created_at = excluded.created_at,
    updated_at = excluded.updated_at,
    start = excluded.start,
    end = excluded.end,
    timezone_offset = excluded.timezone_offset,
    nap = excluded.nap,
    score_state = excluded.score_state,
    score_json = excluded.score_json,
    fetched_at = CURRENT_TIMESTAMP;

-- name: GetSleep :one
SELECT * FROM sleeps WHERE id = ?;

-- name: GetSleepByCycleID :one
SELECT * FROM sleeps WHERE cycle_id = ? AND nap = 0 LIMIT 1;

-- name: GetSleepsByDateRange :many
SELECT * FROM sleeps
WHERE start >= sqlc.arg(range_start) AND start <= sqlc.arg(range_end)
ORDER BY start DESC
LIMIT sqlc.arg(limit);

-- name: GetSleepsByDateRangeCursor :many
SELECT * FROM sleeps
WHERE start >= sqlc.arg(range_start) AND start <= sqlc.arg(range_end) AND start < sqlc.arg(cursor)
ORDER BY start DESC
LIMIT sqlc.arg(limit);

-- name: GetNapsByCycleID :many
SELECT * FROM sleeps WHERE cycle_id = ? AND nap = 1 ORDER BY start DESC;

-- name: DeleteSleep :exec
DELETE FROM sleeps WHERE id = ?;
