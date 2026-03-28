-- +goose Up
CREATE TABLE IF NOT EXISTS dead_letter_queue (
    id              TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    job_id          TEXT NOT NULL,
    pipeline_run_id TEXT NOT NULL,
    failure_reason  TEXT NOT NULL DEFAULT '',
    retry_count     INTEGER NOT NULL DEFAULT 0,
    max_retries     INTEGER NOT NULL DEFAULT 3,
    job_data        TEXT NOT NULL DEFAULT '{}',
    status          TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','retried','purged')),
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_dlq_status ON dead_letter_queue(status);
CREATE INDEX idx_dlq_pipeline_run ON dead_letter_queue(pipeline_run_id);

-- +goose Down
DROP TABLE IF EXISTS dead_letter_queue;
