-- +goose Up
-- In-app notification center
CREATE TABLE IF NOT EXISTS in_app_notifications (
    id          TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title       TEXT NOT NULL,
    message     TEXT NOT NULL DEFAULT '',
    type        TEXT NOT NULL DEFAULT 'info' CHECK(type IN ('info','success','warning','error')),
    category    TEXT NOT NULL DEFAULT 'system' CHECK(category IN ('system','pipeline','deployment','approval','agent','security')),
    link        TEXT NOT NULL DEFAULT '',
    is_read     BOOLEAN NOT NULL DEFAULT 0,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_in_app_notifications_user ON in_app_notifications(user_id, is_read);
CREATE INDEX IF NOT EXISTS idx_in_app_notifications_created ON in_app_notifications(created_at);

-- User notification preferences
CREATE TABLE IF NOT EXISTS notification_preferences (
    id          TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    user_id     TEXT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    email_enabled       BOOLEAN NOT NULL DEFAULT 1,
    in_app_enabled      BOOLEAN NOT NULL DEFAULT 1,
    pipeline_success    BOOLEAN NOT NULL DEFAULT 1,
    pipeline_failure    BOOLEAN NOT NULL DEFAULT 1,
    deployment_success  BOOLEAN NOT NULL DEFAULT 1,
    deployment_failure  BOOLEAN NOT NULL DEFAULT 1,
    approval_requested  BOOLEAN NOT NULL DEFAULT 1,
    approval_resolved   BOOLEAN NOT NULL DEFAULT 1,
    agent_offline       BOOLEAN NOT NULL DEFAULT 1,
    security_alerts     BOOLEAN NOT NULL DEFAULT 1,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
