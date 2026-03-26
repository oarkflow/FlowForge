-- +goose Up
ALTER TABLE pipeline_runs ADD COLUMN deploy_url TEXT;

-- +goose Down
ALTER TABLE pipeline_runs DROP COLUMN deploy_url;
