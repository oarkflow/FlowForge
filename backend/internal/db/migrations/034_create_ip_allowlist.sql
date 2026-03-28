-- +goose Up
CREATE TABLE ip_allowlist (
    id          TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    project_id  TEXT REFERENCES projects(id) ON DELETE CASCADE,
    scope       TEXT NOT NULL DEFAULT 'global' CHECK(scope IN ('global','project')),
    cidr        TEXT NOT NULL,
    label       TEXT NOT NULL DEFAULT '',
    created_by  TEXT REFERENCES users(id),
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_ip_allowlist_scope ON ip_allowlist(scope);
CREATE INDEX idx_ip_allowlist_project ON ip_allowlist(project_id);

-- +goose Down
DROP TABLE IF EXISTS ip_allowlist;
