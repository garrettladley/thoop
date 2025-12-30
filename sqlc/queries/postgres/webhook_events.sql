-- name: InsertWebhookEvent :one
INSERT INTO webhook_events (trace_id, whoop_user_id, timestamp, entity_id, entity_type, action)
VALUES (sqlc.arg(trace_id), sqlc.arg(whoop_user_id), sqlc.arg(timestamp), sqlc.arg(entity_id), sqlc.arg(entity_type), sqlc.arg(action))
ON CONFLICT (trace_id) DO NOTHING
RETURNING id;

-- name: GetUnackedWebhookEvents :many
SELECT id, trace_id, whoop_user_id, timestamp, entity_id, entity_type, action
FROM webhook_events
WHERE whoop_user_id = sqlc.arg(whoop_user_id)
  AND acknowledged_at IS NULL
  AND id > sqlc.arg(cursor)
ORDER BY id
LIMIT sqlc.arg(max_results);

-- name: AcknowledgeWebhookEventsByTraceIDs :exec
UPDATE webhook_events
SET acknowledged_at = now()
WHERE whoop_user_id = sqlc.arg(whoop_user_id)
  AND trace_id = ANY(sqlc.arg(trace_ids)::text[])
  AND acknowledged_at IS NULL;
