-- +goose Up
ALTER TABLE projects ADD COLUMN primary_branch TEXT NOT NULL DEFAULT 'main';

-- +goose Down
ALTER TABLE projects DROP COLUMN primary_branch;
