-- +goose Up
CREATE TABLE repositories (
    id              TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    project_id      TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    provider        TEXT NOT NULL CHECK(provider IN ('github','gitlab','bitbucket')),
    provider_id     TEXT NOT NULL,
    full_name       TEXT NOT NULL,
    clone_url       TEXT NOT NULL,
    ssh_url         TEXT,
    default_branch  TEXT NOT NULL DEFAULT 'main',
    webhook_id      TEXT,
    webhook_secret  TEXT,
    access_token_enc TEXT,
    ssh_key_enc     TEXT,
    is_active       INTEGER NOT NULL DEFAULT 1,
    last_sync_at    DATETIME,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE IF EXISTS repositories;
