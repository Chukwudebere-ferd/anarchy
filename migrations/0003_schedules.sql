ALTER TABLE deployments ADD COLUMN IF NOT EXISTS schedule TEXT NOT NULL DEFAULT '@every 2h';
ALTER TABLE deployments ADD COLUMN IF NOT EXISTS jitter_seconds INT NOT NULL DEFAULT 600;

CREATE TABLE IF NOT EXISTS tasks (
  id TEXT PRIMARY KEY,
  deployment_id TEXT NOT NULL REFERENCES deployments(deployment_id),
  type TEXT NOT NULL,
  payload TEXT NOT NULL DEFAULT '',
  schedule TEXT NOT NULL DEFAULT '',
  needs_approval BOOLEAN NOT NULL DEFAULT FALSE,
  approved_by TEXT NOT NULL DEFAULT '',
  approved_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_tasks_deployment ON tasks(deployment_id);
