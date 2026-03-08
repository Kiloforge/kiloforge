-- +goose Up
ALTER TABLE agents ADD COLUMN finished_at TEXT;

-- +goose Down
-- SQLite does not support DROP COLUMN before 3.35.0; no-op for safety.
