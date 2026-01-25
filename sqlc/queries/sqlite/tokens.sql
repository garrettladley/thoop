-- name: GetTokenMetadata :one
SELECT * FROM tokens WHERE id = 1;

-- name: UpsertTokenMetadata :exec
INSERT INTO tokens (id, token_type, expiry)
VALUES (1, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    token_type = excluded.token_type,
    expiry = excluded.expiry;

-- name: DeleteTokenMetadata :exec
DELETE FROM tokens WHERE id = 1;
