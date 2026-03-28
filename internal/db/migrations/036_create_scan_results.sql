-- +goose Up
CREATE TABLE IF NOT EXISTS scan_results (
    id              TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    run_id          TEXT NOT NULL REFERENCES pipeline_runs(id) ON DELETE CASCADE,
    scanner_type    TEXT NOT NULL CHECK(scanner_type IN ('trivy','grype')),
    target          TEXT NOT NULL DEFAULT '',
    vulnerabilities TEXT NOT NULL DEFAULT '[]',
    critical_count  INTEGER NOT NULL DEFAULT 0,
    high_count      INTEGER NOT NULL DEFAULT 0,
    medium_count    INTEGER NOT NULL DEFAULT 0,
    low_count       INTEGER NOT NULL DEFAULT 0,
    status          TEXT NOT NULL DEFAULT 'pass' CHECK(status IN ('pass','fail','error')),
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_scan_results_run_id ON scan_results(run_id);

-- +goose Down
DROP TABLE IF EXISTS scan_results;
