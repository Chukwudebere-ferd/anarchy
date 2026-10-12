package training

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SaveDeployments persists deployments in chunks of 100 rows per statement.
func SaveDeployments(ctx context.Context, pool *pgxpool.Pool, deps []Deployment) error {
	for start := 0; start < len(deps); start += 100 {
		end := start + 100
		if end > len(deps) {
			end = len(deps)
		}
		chunk := deps[start:end]
		placeholders := make([]string, 0, len(chunk))
		args := make([]any, 0, len(chunk)*5)
		for i, d := range chunk {
			base := i * 5
			placeholders = append(placeholders, fmt.Sprintf("($%d,$%d,$%d,$%d,$%d)",
				base+1, base+2, base+3, base+4, base+5))
			args = append(args, d.DeploymentID, d.Identity.ID, d.Character.ID, d.Schedule, d.JitterSeconds)
		}
		q := `INSERT INTO deployments (deployment_id, identity_id, character_id, schedule, jitter_seconds)
VALUES ` + strings.Join(placeholders, ",") + `
ON CONFLICT (deployment_id) DO UPDATE SET
  identity_id=EXCLUDED.identity_id, character_id=EXCLUDED.character_id,
  schedule=EXCLUDED.schedule, jitter_seconds=EXCLUDED.jitter_seconds`
		if _, err := pool.Exec(ctx, q, args...); err != nil {
			return err
		}
	}
	return nil
}

// ResetDeployments clears tasks and deployments for a clean pilot batch.
// Identities and characters are never touched.
func ResetDeployments(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, `TRUNCATE tasks, deployments`); err != nil {
		return err
	}
	return nil
}

// CountDeployments returns how many deployments are stored.
func CountDeployments(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	var n int
	err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM deployments`).Scan(&n)
	return n, err
}
