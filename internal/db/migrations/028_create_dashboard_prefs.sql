-- +goose Up
-- Dashboard customization preferences
CREATE TABLE IF NOT EXISTS dashboard_preferences (
    id          TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    user_id     TEXT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    layout      TEXT NOT NULL DEFAULT '[]',
    theme       TEXT NOT NULL DEFAULT 'default',
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
