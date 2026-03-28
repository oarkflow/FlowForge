-- +goose Up
ALTER TABLE secrets ADD COLUMN rotation_interval TEXT;
ALTER TABLE secrets ADD COLUMN last_rotated_at DATETIME;
ALTER TABLE secrets ADD COLUMN provider_type TEXT NOT NULL DEFAULT 'local';

-- +goose Down
-- SQLite does not support DROP COLUMN in older versions, so we recreate.
CREATE TABLE secrets_backup AS SELECT id, project_id, org_id, scope, key, value_enc, masked, created_by, created_at, updated_at FROM secrets;
DROP TABLE secrets;
CREATE TABLE secrets (
    id          TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    project_id  TEXT REFERENCES projects(id) ON DELETE CASCADE,
    org_id      TEXT REFERENCES organizations(id) ON DELETE CASCADE,
    scope       TEXT NOT NULL CHECK(scope IN ('project','org','global')),
    key         TEXT NOT NULL,
    value_enc   TEXT NOT NULL,
    masked      INTEGER NOT NULL DEFAULT 1,
    created_by  TEXT REFERENCES users(id),
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO secrets SELECT * FROM secrets_backup;
DROP TABLE secrets_backup;
