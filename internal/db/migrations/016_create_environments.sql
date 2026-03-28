-- +goose Up
CREATE TABLE IF NOT EXISTS environments (
    id              TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    project_id      TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    slug            TEXT NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    url             TEXT NOT NULL DEFAULT '',
    is_production   BOOLEAN NOT NULL DEFAULT 0,
    auto_deploy_branch TEXT NOT NULL DEFAULT '',
    required_approvers TEXT NOT NULL DEFAULT '[]',
    protection_rules TEXT NOT NULL DEFAULT '{}',
    deploy_freeze   BOOLEAN NOT NULL DEFAULT 0,
    lock_owner_id   TEXT,
    lock_reason     TEXT NOT NULL DEFAULT '',
    locked_at       DATETIME,
    current_deployment_id TEXT,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(project_id, slug)
);
CREATE INDEX idx_environments_project_id ON environments(project_id);

-- +goose Down
DROP INDEX IF EXISTS idx_environments_project_id;
DROP TABLE IF EXISTS environments;
