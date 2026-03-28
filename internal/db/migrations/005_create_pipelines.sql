-- +goose Up
CREATE TABLE pipelines (
    id              TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    project_id      TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    repository_id   TEXT REFERENCES repositories(id),
    name            TEXT NOT NULL,
    description     TEXT,
    config_source   TEXT NOT NULL DEFAULT 'db' CHECK(config_source IN ('db','repo')),
    config_path     TEXT DEFAULT '.flowforge.yml',
    config_content  TEXT,
    config_version  INTEGER NOT NULL DEFAULT 1,
    triggers        TEXT NOT NULL DEFAULT '{}',
    is_active       INTEGER NOT NULL DEFAULT 1,
    created_by      TEXT REFERENCES users(id),
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at      DATETIME
);

CREATE TABLE pipeline_versions (
    id          TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    pipeline_id TEXT NOT NULL REFERENCES pipelines(id) ON DELETE CASCADE,
    version     INTEGER NOT NULL,
    config      TEXT NOT NULL,
    message     TEXT,
    created_by  TEXT REFERENCES users(id),
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(pipeline_id, version)
);

-- +goose Down
DROP TABLE IF EXISTS pipeline_versions;
DROP TABLE IF EXISTS pipelines;
