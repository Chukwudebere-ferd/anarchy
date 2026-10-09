package training

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SaveDeployment persists one randomized deployment with its schedule.
func SaveDeployment(ctx context.Context, pool *pgxpool.Pool, d Deployment) error {
	_, err := pool.Exec(ctx, `
INSERT INTO deployments (deployment_id, identity_id, character_id, schedule, jitter_seconds)
VALUES ($1,$2,$3,$4,$5)
ON CONFLICT (deployment_id) DO UPDATE SET
  identity_id=EXCLUDED.identity_id, character_id=EXCLUDED.character_id,
  schedule=EXCLUDED.schedule, jitter_seconds=EXCLUDED.jitter_seconds
`, d.DeploymentID, d.Identity.ID, d.Character.ID, d.Schedule, d.JitterSeconds)
	return err
}

// CountDeployments returns how many deployments are stored.
func CountDeployments(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	var n int
	err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM deployments`).Scan(&n)
	return n, err
}
