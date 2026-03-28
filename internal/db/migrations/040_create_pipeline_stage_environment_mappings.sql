-- +goose Up
CREATE TABLE pipeline_stage_environment_mappings (
    id              TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    project_id      TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    pipeline_id     TEXT NOT NULL REFERENCES pipelines(id) ON DELETE CASCADE,
    stage_name      TEXT NOT NULL,
    environment_id  TEXT NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(pipeline_id, stage_name)
);

CREATE INDEX idx_stage_env_mappings_project_pipeline
    ON pipeline_stage_environment_mappings(project_id, pipeline_id);

-- +goose Down
DROP INDEX IF EXISTS idx_stage_env_mappings_project_pipeline;
DROP TABLE IF EXISTS pipeline_stage_environment_mappings;
