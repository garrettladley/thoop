-- name: CreateUser :one
INSERT INTO users (whoop_user_id)
VALUES ($1)
RETURNING *;

-- name: GetOrCreateUser :one
INSERT INTO users (whoop_user_id)
VALUES ($1)
ON CONFLICT (whoop_user_id) DO UPDATE SET whoop_user_id = EXCLUDED.whoop_user_id
RETURNING *;

-- name: GetUser :one
SELECT * FROM users WHERE whoop_user_id = $1;

-- name: BanUser :exec
UPDATE users SET banned = true WHERE whoop_user_id = $1;

-- name: UnbanUser :exec
UPDATE users SET banned = false WHERE whoop_user_id = $1;
