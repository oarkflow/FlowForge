-- +goose Up
-- Expand the provider CHECK constraint to include 'git', 'local', 'upload'.
-- SQLite does not support ALTER CONSTRAINT, so we recreate the table.

CREATE TABLE repositories_new (
    id              TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    project_id      TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    provider        TEXT NOT NULL CHECK(provider IN ('github','gitlab','bitbucket','git','local','upload')),
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

INSERT INTO repositories_new SELECT * FROM repositories;
DROP TABLE repositories;
ALTER TABLE repositories_new RENAME TO repositories;

-- +goose Down
CREATE TABLE repositories_old (
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

INSERT INTO repositories_old SELECT * FROM repositories WHERE provider IN ('github','gitlab','bitbucket');
DROP TABLE repositories;
ALTER TABLE repositories_old RENAME TO repositories;
