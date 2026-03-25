-- +goose Up
CREATE TABLE agents (
    id          TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    name        TEXT NOT NULL,
    token_hash  TEXT NOT NULL UNIQUE,
    labels      TEXT NOT NULL DEFAULT '{}',
    executor    TEXT NOT NULL DEFAULT 'local' CHECK(executor IN ('local','docker','kubernetes')),
    status      TEXT NOT NULL DEFAULT 'offline' CHECK(status IN ('online','offline','busy','draining')),
    version     TEXT,
    os          TEXT,
    arch        TEXT,
    cpu_cores   INTEGER,
    memory_mb   INTEGER,
    ip_address  TEXT,
    last_seen_at DATETIME,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE IF EXISTS agents;
