-- +goose Up
-- Monorepo support: path-based pipeline filtering
ALTER TABLE pipelines ADD COLUMN path_filters TEXT NOT NULL DEFAULT '';
ALTER TABLE pipelines ADD COLUMN ignore_paths TEXT NOT NULL DEFAULT '';
