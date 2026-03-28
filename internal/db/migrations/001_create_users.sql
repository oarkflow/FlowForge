-- +goose Up
CREATE TABLE users (
    id          TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    email       TEXT NOT NULL UNIQUE,
    username    TEXT NOT NULL UNIQUE,
    password_hash TEXT,
    display_name TEXT,
    avatar_url  TEXT,
    role        TEXT NOT NULL DEFAULT 'viewer' CHECK(role IN ('owner','admin','developer','viewer')),
    totp_secret TEXT,
    totp_enabled INTEGER NOT NULL DEFAULT 0,
    is_active   INTEGER NOT NULL DEFAULT 1,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at  DATETIME
);

-- +goose Down
DROP TABLE IF EXISTS users;
