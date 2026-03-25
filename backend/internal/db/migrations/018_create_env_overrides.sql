-- +goose Up
CREATE TABLE IF NOT EXISTS env_overrides (
    id              TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    environment_id  TEXT NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    key             TEXT NOT NULL,
    value_enc       TEXT NOT NULL DEFAULT '',
    is_secret       BOOLEAN NOT NULL DEFAULT 0,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(environment_id, key)
);
CREATE INDEX idx_env_overrides_environment_id ON env_overrides(environment_id);

-- +goose Down
DROP INDEX IF EXISTS idx_env_overrides_environment_id;
DROP TABLE IF EXISTS env_overrides;
