-- +goose Up
CREATE TABLE queue_items (
    track_id TEXT PRIMARY KEY,
    project_slug TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'queued',
    agent_id TEXT,
    enqueued_at TEXT NOT NULL,
    assigned_at TEXT,
    completed_at TEXT
);
CREATE INDEX idx_queue_items_status ON queue_items(status);

-- +goose Down
DROP INDEX IF EXISTS idx_queue_items_status;
DROP TABLE IF EXISTS queue_items;
