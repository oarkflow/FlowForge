-- +goose Up

-- Add strategy fields to environments
ALTER TABLE environments ADD COLUMN strategy TEXT NOT NULL DEFAULT 'recreate' CHECK(strategy IN ('recreate','rolling','blue_green','canary'));
ALTER TABLE environments ADD COLUMN strategy_config TEXT NOT NULL DEFAULT '{}';
ALTER TABLE environments ADD COLUMN health_check_url TEXT NOT NULL DEFAULT '';
ALTER TABLE environments ADD COLUMN health_check_interval INTEGER NOT NULL DEFAULT 30;
ALTER TABLE environments ADD COLUMN health_check_timeout INTEGER NOT NULL DEFAULT 10;
ALTER TABLE environments ADD COLUMN health_check_retries INTEGER NOT NULL DEFAULT 3;
ALTER TABLE environments ADD COLUMN health_check_path TEXT NOT NULL DEFAULT '/health';
ALTER TABLE environments ADD COLUMN health_check_expected_status INTEGER NOT NULL DEFAULT 200;

-- Add strategy tracking to deployments
ALTER TABLE deployments ADD COLUMN strategy TEXT NOT NULL DEFAULT 'recreate';
ALTER TABLE deployments ADD COLUMN canary_weight INTEGER NOT NULL DEFAULT 0;
ALTER TABLE deployments ADD COLUMN health_check_results TEXT NOT NULL DEFAULT '[]';
ALTER TABLE deployments ADD COLUMN strategy_state TEXT NOT NULL DEFAULT '{}';

-- +goose Down

-- SQLite does not support DROP COLUMN prior to 3.35.0;
-- for safety these are no-ops in the down migration.
