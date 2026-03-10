-- +goose Up
ALTER TABLE projects ADD COLUMN mirror_dir TEXT;

-- +goose Down
ALTER TABLE projects DROP COLUMN mirror_dir;
