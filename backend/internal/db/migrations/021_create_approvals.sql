-- +goose Up
CREATE TABLE IF NOT EXISTS approvals (
    id                TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    type              TEXT NOT NULL CHECK(type IN ('deployment','pipeline_run')),
    deployment_id     TEXT REFERENCES deployments(id) ON DELETE CASCADE,
    pipeline_run_id   TEXT REFERENCES pipeline_runs(id) ON DELETE CASCADE,
    environment_id    TEXT REFERENCES environments(id) ON DELETE CASCADE,
    project_id        TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    requested_by      TEXT NOT NULL,
    status            TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','approved','rejected','expired','cancelled')),
    required_approvers TEXT NOT NULL DEFAULT '[]',
    min_approvals     INTEGER NOT NULL DEFAULT 1,
    current_approvals INTEGER NOT NULL DEFAULT 0,
    expires_at        DATETIME,
    resolved_at       DATETIME,
    created_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_approvals_status ON approvals(status);
CREATE INDEX IF NOT EXISTS idx_approvals_deployment_id ON approvals(deployment_id);
CREATE INDEX IF NOT EXISTS idx_approvals_pipeline_run_id ON approvals(pipeline_run_id);
CREATE INDEX IF NOT EXISTS idx_approvals_project_id ON approvals(project_id);

-- +goose Down
DROP INDEX IF EXISTS idx_approvals_project_id;
DROP INDEX IF EXISTS idx_approvals_pipeline_run_id;
DROP INDEX IF EXISTS idx_approvals_deployment_id;
DROP INDEX IF EXISTS idx_approvals_status;
DROP TABLE IF EXISTS approvals;
