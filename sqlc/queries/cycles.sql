-- name: UpsertCycle :exec
INSERT INTO cycles (id, user_id, created_at, updated_at, start, end, timezone_offset, score_state, score_json, fetched_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(id) DO UPDATE SET
    user_id = excluded.user_id,
    created_at = excluded.created_at,
    updated_at = excluded.updated_at,
    start = excluded.start,
    end = excluded.end,
    timezone_offset = excluded.timezone_offset,
    score_state = excluded.score_state,
    score_json = excluded.score_json,
    fetched_at = CURRENT_TIMESTAMP;

-- name: GetCycle :one
SELECT * FROM cycles WHERE id = ?;

-- name: GetLatestCycles :many
SELECT * FROM cycles ORDER BY start DESC LIMIT ?;

-- name: GetCyclesByDateRange :many
SELECT * FROM cycles
WHERE start >= sqlc.arg(range_start) AND start <= sqlc.arg(range_end)
ORDER BY start DESC
LIMIT sqlc.arg(limit);

-- name: GetCyclesByDateRangeCursor :many
SELECT * FROM cycles
WHERE start >= sqlc.arg(range_start) AND start <= sqlc.arg(range_end) AND start < sqlc.arg(cursor)
ORDER BY start DESC
LIMIT sqlc.arg(limit);

-- name: GetPendingCycles :many
SELECT * FROM cycles WHERE score_state = 'PENDING_SCORE' ORDER BY start DESC;

-- name: DeleteCycle :exec
DELETE FROM cycles WHERE id = ?;
