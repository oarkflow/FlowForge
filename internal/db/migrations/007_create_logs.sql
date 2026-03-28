-- +goose Up
CREATE TABLE run_logs (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id      TEXT NOT NULL REFERENCES pipeline_runs(id) ON DELETE CASCADE,
    step_run_id TEXT REFERENCES step_runs(id) ON DELETE CASCADE,
    stream      TEXT NOT NULL DEFAULT 'stdout' CHECK(stream IN ('stdout','stderr','system')),
    content     TEXT NOT NULL,
    ts          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_run_logs_run_id ON run_logs(run_id);
CREATE INDEX idx_run_logs_step_id ON run_logs(step_run_id);

-- +goose Down
DROP INDEX IF EXISTS idx_run_logs_step_id;
DROP INDEX IF EXISTS idx_run_logs_run_id;
DROP TABLE IF EXISTS run_logs;
