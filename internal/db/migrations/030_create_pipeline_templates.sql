-- +goose Up
CREATE TABLE IF NOT EXISTS pipeline_templates (
    id          TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    category    TEXT NOT NULL DEFAULT 'general',
    config      TEXT NOT NULL,
    is_builtin  INTEGER NOT NULL DEFAULT 0,
    is_public   INTEGER NOT NULL DEFAULT 1,
    author      TEXT NOT NULL DEFAULT 'system',
    downloads   INTEGER NOT NULL DEFAULT 0,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_pipeline_templates_category ON pipeline_templates(category);
CREATE INDEX idx_pipeline_templates_is_builtin ON pipeline_templates(is_builtin);

-- +goose Down
DROP TABLE IF EXISTS pipeline_templates;
