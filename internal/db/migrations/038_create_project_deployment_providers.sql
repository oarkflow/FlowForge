-- +goose Up
CREATE TABLE project_deployment_providers (
    id              TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    project_id      TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    provider_type   TEXT NOT NULL,
    config_enc      TEXT NOT NULL,
    is_active       INTEGER NOT NULL DEFAULT 1,
    created_by      TEXT REFERENCES users(id),
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(project_id, name)
);

CREATE INDEX idx_project_deployment_providers_project_id
    ON project_deployment_providers(project_id);

-- +goose Down
DROP INDEX IF EXISTS idx_project_deployment_providers_project_id;
DROP TABLE IF EXISTS project_deployment_providers;
