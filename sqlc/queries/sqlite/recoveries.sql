-- name: UpsertRecovery :exec
INSERT INTO recoveries (cycle_id, sleep_id, user_id, created_at, updated_at, score_state, score_json, fetched_at)
VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(cycle_id) DO UPDATE SET
    sleep_id = excluded.sleep_id,
    user_id = excluded.user_id,
    created_at = excluded.created_at,
    updated_at = excluded.updated_at,
    score_state = excluded.score_state,
    score_json = excluded.score_json,
    fetched_at = CURRENT_TIMESTAMP;

-- name: GetRecovery :one
SELECT * FROM recoveries WHERE cycle_id = ?;

-- name: GetRecoveriesByCycleIDs :many
SELECT * FROM recoveries WHERE cycle_id IN (sqlc.slice('cycle_ids'));

-- name: DeleteRecovery :exec
DELETE FROM recoveries WHERE cycle_id = ?;
