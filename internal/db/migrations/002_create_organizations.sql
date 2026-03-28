-- +goose Up
CREATE TABLE organizations (
    id          TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL UNIQUE,
    logo_url    TEXT,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE org_members (
    org_id      TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role        TEXT NOT NULL DEFAULT 'developer',
    joined_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (org_id, user_id)
);

-- +goose Down
DROP TABLE IF EXISTS org_members;
DROP TABLE IF EXISTS organizations;
