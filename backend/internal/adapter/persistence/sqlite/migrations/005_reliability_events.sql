-- +goose Up
CREATE TABLE reliability_events (
    id         TEXT PRIMARY KEY,
    event_type TEXT NOT NULL,
    severity   TEXT NOT NULL,
    agent_id   TEXT,
    scope      TEXT,
    detail     TEXT,
    created_at TEXT NOT NULL
);
CREATE INDEX idx_reliability_events_type_created ON reliability_events(event_type, created_at);
CREATE INDEX idx_reliability_events_agent ON reliability_events(agent_id);
CREATE INDEX idx_reliability_events_created ON reliability_events(created_at);

-- +goose Down
DROP TABLE IF EXISTS reliability_events;
