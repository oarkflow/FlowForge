-- +goose Up
CREATE TABLE IF NOT EXISTS approval_responses (
    id            TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    approval_id   TEXT NOT NULL REFERENCES approvals(id) ON DELETE CASCADE,
    approver_id   TEXT NOT NULL,
    approver_name TEXT NOT NULL DEFAULT '',
    decision      TEXT NOT NULL CHECK(decision IN ('approve','reject')),
    comment       TEXT NOT NULL DEFAULT '',
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_approval_responses_approval_id ON approval_responses(approval_id);

-- +goose Down
DROP INDEX IF EXISTS idx_approval_responses_approval_id;
DROP TABLE IF EXISTS approval_responses;
