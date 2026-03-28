-- +goose Up
ALTER TABLE projects ADD COLUMN log_retention_days INTEGER NOT NULL DEFAULT 0;

-- +goose Down
-- SQLite does not support DROP COLUMN in older versions; this is a no-op.
