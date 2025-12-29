-- name: GetToken :one
SELECT * FROM tokens WHERE id = 1;

-- name: UpsertToken :exec
INSERT INTO tokens (id, access_token, refresh_token, token_type, expiry)
VALUES (1, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    access_token = excluded.access_token,
    refresh_token = excluded.refresh_token,
    token_type = excluded.token_type,
    expiry = excluded.expiry;

-- name: DeleteToken :exec
DELETE FROM tokens WHERE id = 1;
