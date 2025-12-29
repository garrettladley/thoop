-- name: CreateAPIKey :one
INSERT INTO api_keys (whoop_user_id, key_hash, name)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetAPIKeyByHash :one
SELECT * FROM api_keys WHERE key_hash = $1;

-- name: GetAPIKeysByUser :many
SELECT * FROM api_keys WHERE whoop_user_id = $1 ORDER BY created_at DESC;

-- name: UpdateAPIKeyLastUsed :exec
UPDATE api_keys SET last_used_at = now() WHERE id = $1;

-- name: RevokeAPIKey :exec
UPDATE api_keys SET revoked = true WHERE id = $1;

-- name: DeleteAPIKey :exec
DELETE FROM api_keys WHERE id = $1;
