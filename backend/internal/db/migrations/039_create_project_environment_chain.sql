-- +goose Up
CREATE TABLE project_environment_chain (
    id                  TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    project_id          TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    source_environment_id TEXT NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    target_environment_id TEXT NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    position            INTEGER NOT NULL DEFAULT 0,
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(project_id, source_environment_id, target_environment_id)
);

CREATE INDEX idx_project_environment_chain_project_id
    ON project_environment_chain(project_id);

CREATE INDEX idx_project_environment_chain_source_target
    ON project_environment_chain(project_id, source_environment_id, target_environment_id);

-- +goose Down
DROP INDEX IF EXISTS idx_project_environment_chain_source_target;
DROP INDEX IF EXISTS idx_project_environment_chain_project_id;
DROP TABLE IF EXISTS project_environment_chain;
