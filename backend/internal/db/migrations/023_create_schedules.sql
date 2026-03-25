-- +goose Up
CREATE TABLE IF NOT EXISTS pipeline_schedules (
    id              TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    pipeline_id     TEXT NOT NULL REFERENCES pipelines(id) ON DELETE CASCADE,
    project_id      TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    cron_expression TEXT NOT NULL,
    timezone        TEXT NOT NULL DEFAULT 'UTC',
    description     TEXT NOT NULL DEFAULT '',
    enabled         BOOLEAN NOT NULL DEFAULT 1,
    branch          TEXT NOT NULL DEFAULT 'main',
    environment_id  TEXT REFERENCES environments(id) ON DELETE SET NULL,
    variables       TEXT NOT NULL DEFAULT '{}',
    next_run_at     DATETIME,
    last_run_at     DATETIME,
    last_run_status TEXT NOT NULL DEFAULT '',
    last_run_id     TEXT,
    run_count       INTEGER NOT NULL DEFAULT 0,
    created_by      TEXT NOT NULL DEFAULT '',
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_pipeline_schedules_pipeline_id ON pipeline_schedules(pipeline_id);
CREATE INDEX idx_pipeline_schedules_next_run ON pipeline_schedules(enabled, next_run_at);

-- +goose Down
DROP INDEX IF EXISTS idx_pipeline_schedules_next_run;
DROP INDEX IF EXISTS idx_pipeline_schedules_pipeline_id;
DROP TABLE IF EXISTS pipeline_schedules;
