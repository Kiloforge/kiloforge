-- +goose Up
ALTER TABLE agents ADD COLUMN name TEXT NOT NULL DEFAULT '';

-- +goose Down
-- SQLite does not support DROP COLUMN before 3.35.0; recreate without name.
CREATE TABLE agents_backup AS SELECT
	id, role, ref, status, session_id, pid, worktree_dir, log_file,
	started_at, updated_at, suspended_at, shutdown_reason, resume_error, model
FROM agents;
DROP TABLE agents;
ALTER TABLE agents_backup RENAME TO agents;
CREATE INDEX idx_agents_status ON agents(status);
CREATE INDEX idx_agents_ref ON agents(ref);
