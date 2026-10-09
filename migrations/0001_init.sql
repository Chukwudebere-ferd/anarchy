CREATE TABLE IF NOT EXISTS identities (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  email_enc BYTEA,
  phone_enc BYTEA,
  profile TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  disabled BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS characters (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  version INT NOT NULL,
  traits JSONB NOT NULL DEFAULT '[]',
  communication_style TEXT NOT NULL DEFAULT '',
  goals JSONB NOT NULL DEFAULT '[]',
  knowledge JSONB NOT NULL DEFAULT '[]',
  boundaries JSONB NOT NULL DEFAULT '[]',
  examples JSONB NOT NULL DEFAULT '[]',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (name, version)
);

CREATE TABLE IF NOT EXISTS deployments (
  deployment_id TEXT PRIMARY KEY,
  identity_id TEXT NOT NULL REFERENCES identities(id),
  character_id TEXT NOT NULL REFERENCES characters(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS eval_results (
  id TEXT PRIMARY KEY,
  scenario_id TEXT NOT NULL,
  character_id TEXT NOT NULL,
  character_version INT NOT NULL,
  passed BOOLEAN NOT NULL,
  score DOUBLE PRECISION NOT NULL,
  notes TEXT NOT NULL DEFAULT '',
  evaluated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS audit_logs (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  deployment_id TEXT NOT NULL,
  action TEXT NOT NULL,
  tool TEXT NOT NULL DEFAULT '',
  approved BOOLEAN NOT NULL DEFAULT FALSE,
  meta JSONB NOT NULL DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_deployments_identity ON deployments(identity_id);
CREATE INDEX IF NOT EXISTS idx_deployments_character ON deployments(character_id);
CREATE INDEX IF NOT EXISTS idx_eval_character ON eval_results(character_id, character_version);
CREATE INDEX IF NOT EXISTS idx_audit_deployment ON audit_logs(deployment_id);
