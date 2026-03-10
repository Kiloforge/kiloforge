-- +goose Up
CREATE TABLE notifications (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL,
    title TEXT NOT NULL,
    body TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    acknowledged_at TEXT
);
CREATE INDEX idx_notifications_agent ON notifications(agent_id);
CREATE INDEX idx_notifications_created ON notifications(created_at);

-- +goose Down
DROP INDEX IF EXISTS idx_notifications_created;
DROP INDEX IF EXISTS idx_notifications_agent;
DROP TABLE IF EXISTS notifications;
