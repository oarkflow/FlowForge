-- +goose Up
CREATE TABLE IF NOT EXISTS deployments (
    id                TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    environment_id    TEXT NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    pipeline_run_id   TEXT REFERENCES pipeline_runs(id),
    version           TEXT NOT NULL DEFAULT '',
    status            TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','deploying','live','failed','rolled_back')),
    commit_sha        TEXT NOT NULL DEFAULT '',
    image_tag         TEXT NOT NULL DEFAULT '',
    deployed_by       TEXT NOT NULL DEFAULT '',
    started_at        DATETIME,
    finished_at       DATETIME,
    health_check_status TEXT NOT NULL DEFAULT 'unknown',
    rollback_from_id  TEXT,
    metadata          TEXT NOT NULL DEFAULT '{}',
    created_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_deployments_environment_id ON deployments(environment_id);
CREATE INDEX idx_deployments_pipeline_run_id ON deployments(pipeline_run_id);

-- +goose Down
DROP INDEX IF EXISTS idx_deployments_pipeline_run_id;
DROP INDEX IF EXISTS idx_deployments_environment_id;
DROP TABLE IF EXISTS deployments;
