-- +goose Up
CREATE TABLE IF NOT EXISTS feature_flags (
    id                  TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    name                TEXT NOT NULL UNIQUE,
    description         TEXT NOT NULL DEFAULT '',
    enabled             INTEGER NOT NULL DEFAULT 0,
    rollout_percentage  INTEGER NOT NULL DEFAULT 100,
    target_users        TEXT NOT NULL DEFAULT '[]',
    target_orgs         TEXT NOT NULL DEFAULT '[]',
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX idx_feature_flags_name ON feature_flags(name);

-- +goose Down
DROP TABLE IF EXISTS feature_flags;
