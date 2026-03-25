-- +goose Up
CREATE TABLE pipeline_runs (
    id              TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    pipeline_id     TEXT NOT NULL REFERENCES pipelines(id) ON DELETE CASCADE,
    number          INTEGER NOT NULL,
    status          TEXT NOT NULL DEFAULT 'queued'
                    CHECK(status IN ('queued','pending','running','success','failure','cancelled','skipped','waiting_approval')),
    trigger_type    TEXT NOT NULL CHECK(trigger_type IN ('push','pull_request','schedule','manual','api','pipeline')),
    trigger_data    TEXT,
    commit_sha      TEXT,
    commit_message  TEXT,
    branch          TEXT,
    tag             TEXT,
    author          TEXT,
    started_at      DATETIME,
    finished_at     DATETIME,
    duration_ms     INTEGER,
    created_by      TEXT REFERENCES users(id),
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(pipeline_id, number)
);

CREATE TABLE stage_runs (
    id          TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    run_id      TEXT NOT NULL REFERENCES pipeline_runs(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    status      TEXT NOT NULL DEFAULT 'pending',
    position    INTEGER NOT NULL,
    started_at  DATETIME,
    finished_at DATETIME
);

CREATE TABLE job_runs (
    id              TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    stage_run_id    TEXT NOT NULL REFERENCES stage_runs(id) ON DELETE CASCADE,
    run_id          TEXT NOT NULL REFERENCES pipeline_runs(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending',
    agent_id        TEXT,
    executor_type   TEXT NOT NULL DEFAULT 'local',
    started_at      DATETIME,
    finished_at     DATETIME
);

CREATE TABLE step_runs (
    id              TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    job_run_id      TEXT NOT NULL REFERENCES job_runs(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending',
    exit_code       INTEGER,
    error_message   TEXT,
    started_at      DATETIME,
    finished_at     DATETIME,
    duration_ms     INTEGER
);

-- +goose Down
DROP TABLE IF EXISTS step_runs;
DROP TABLE IF EXISTS job_runs;
DROP TABLE IF EXISTS stage_runs;
DROP TABLE IF EXISTS pipeline_runs;
