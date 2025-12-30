CREATE TABLE webhook_events (
    trace_id TEXT PRIMARY KEY,
    id BIGINT GENERATED ALWAYS AS IDENTITY,
    whoop_user_id BIGINT NOT NULL REFERENCES users(whoop_user_id) ON DELETE CASCADE,
    timestamp TIMESTAMPTZ NOT NULL,
    entity_id TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    action TEXT NOT NULL,
    acknowledged_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_webhook_events_id ON webhook_events (id);

CREATE INDEX idx_webhook_events_unacked
    ON webhook_events (whoop_user_id, id)
    WHERE acknowledged_at IS NULL;
