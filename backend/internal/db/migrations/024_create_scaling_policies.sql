-- +goose Up
-- Scaling policies for agent auto-scaling
CREATE TABLE IF NOT EXISTS scaling_policies (
    id              TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    name            TEXT NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    enabled         BOOLEAN NOT NULL DEFAULT 1,
    executor_type   TEXT NOT NULL DEFAULT 'docker' CHECK(executor_type IN ('local','docker','kubernetes')),
    labels          TEXT NOT NULL DEFAULT '',
    min_agents      INTEGER NOT NULL DEFAULT 1,
    max_agents      INTEGER NOT NULL DEFAULT 10,
    desired_agents  INTEGER NOT NULL DEFAULT 1,
    scale_up_threshold    INTEGER NOT NULL DEFAULT 5,
    scale_down_threshold  INTEGER NOT NULL DEFAULT 0,
    scale_up_step         INTEGER NOT NULL DEFAULT 1,
    scale_down_step       INTEGER NOT NULL DEFAULT 1,
    cooldown_seconds      INTEGER NOT NULL DEFAULT 300,
    last_scale_action     TEXT NOT NULL DEFAULT '',
    last_scale_at         DATETIME,
    queue_depth           INTEGER NOT NULL DEFAULT 0,
    active_agents         INTEGER NOT NULL DEFAULT 0,
    created_at            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Scaling events for audit trail
CREATE TABLE IF NOT EXISTS scaling_events (
    id              TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    policy_id       TEXT NOT NULL REFERENCES scaling_policies(id) ON DELETE CASCADE,
    action          TEXT NOT NULL CHECK(action IN ('scale_up','scale_down','no_action')),
    from_count      INTEGER NOT NULL,
    to_count        INTEGER NOT NULL,
    reason          TEXT NOT NULL DEFAULT '',
    queue_depth     INTEGER NOT NULL DEFAULT 0,
    active_agents   INTEGER NOT NULL DEFAULT 0,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_scaling_events_policy_id ON scaling_events(policy_id);
CREATE INDEX IF NOT EXISTS idx_scaling_events_created_at ON scaling_events(created_at);

-- +goose Down
DROP INDEX IF EXISTS idx_scaling_events_created_at;
DROP INDEX IF EXISTS idx_scaling_events_policy_id;
DROP TABLE IF EXISTS scaling_events;
DROP TABLE IF EXISTS scaling_policies;
