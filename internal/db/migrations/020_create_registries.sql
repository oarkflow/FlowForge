-- +goose Up
CREATE TABLE IF NOT EXISTS registries (
    id              TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    project_id      TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    type            TEXT NOT NULL CHECK(type IN ('dockerhub','ecr','gcr','acr','harbor','ghcr','generic')),
    url             TEXT NOT NULL DEFAULT '',
    username        TEXT NOT NULL DEFAULT '',
    credentials_enc TEXT NOT NULL DEFAULT '',
    is_default      BOOLEAN NOT NULL DEFAULT 0,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(project_id, name)
);
CREATE INDEX idx_registries_project_id ON registries(project_id);

-- +goose Down
DROP INDEX IF EXISTS idx_registries_project_id;
DROP TABLE IF EXISTS registries;
