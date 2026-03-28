-- +goose Up
CREATE TABLE secret_providers (
    id              TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    project_id      TEXT REFERENCES projects(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    provider_type   TEXT NOT NULL CHECK(provider_type IN ('vault','aws','gcp')),
    config_enc      TEXT NOT NULL,
    is_active       INTEGER NOT NULL DEFAULT 1,
    priority        INTEGER NOT NULL DEFAULT 0,
    created_by      TEXT REFERENCES users(id),
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_secret_providers_project ON secret_providers(project_id);

-- +goose Down
DROP TABLE IF EXISTS secret_providers;
