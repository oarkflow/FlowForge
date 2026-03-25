-- +goose Up
CREATE TABLE artifacts (
    id              TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    run_id          TEXT NOT NULL REFERENCES pipeline_runs(id) ON DELETE CASCADE,
    step_run_id     TEXT REFERENCES step_runs(id),
    name            TEXT NOT NULL,
    path            TEXT NOT NULL,
    size_bytes      INTEGER,
    checksum_sha256 TEXT,
    storage_backend TEXT NOT NULL DEFAULT 'local',
    storage_key     TEXT NOT NULL,
    expire_at       DATETIME,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE IF EXISTS artifacts;
