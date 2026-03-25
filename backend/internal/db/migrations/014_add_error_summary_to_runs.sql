-- +goose Up
ALTER TABLE pipeline_runs ADD COLUMN error_summary TEXT;

-- +goose Down
-- SQLite doesn't support DROP COLUMN before 3.35.0, so this is a no-op for older versions.
-- For 3.35.0+:
-- ALTER TABLE pipeline_runs DROP COLUMN error_summary;
