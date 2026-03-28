-- +goose Up
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

-- +goose Down
DROP TABLE IF EXISTS secrets;
