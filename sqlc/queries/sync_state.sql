-- name: GetSyncState :one
SELECT * FROM sync_state WHERE id = 1;

-- name: UpsertSyncState :exec
INSERT INTO sync_state (id, backfill_complete, backfill_watermark, last_full_sync, updated_at)
VALUES (1, ?, ?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(id) DO UPDATE SET
    backfill_complete = excluded.backfill_complete,
    backfill_watermark = excluded.backfill_watermark,
    last_full_sync = excluded.last_full_sync,
    updated_at = CURRENT_TIMESTAMP;

-- name: MarkBackfillComplete :exec
UPDATE sync_state SET backfill_complete = 1, updated_at = CURRENT_TIMESTAMP WHERE id = 1;

-- name: UpdateBackfillWatermark :exec
UPDATE sync_state SET backfill_watermark = ?, updated_at = CURRENT_TIMESTAMP WHERE id = 1;

-- name: UpdateLastFullSync :exec
UPDATE sync_state SET last_full_sync = ?, updated_at = CURRENT_TIMESTAMP WHERE id = 1;
