package character

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Save upserts one character version.
func Save(ctx context.Context, pool *pgxpool.Pool, c Character) error {
	if err := c.Validate(); err != nil {
		return err
	}
	traits, _ := json.Marshal(c.Traits)
	goals, _ := json.Marshal(c.Goals)
	knowledge, _ := json.Marshal(c.Knowledge)
	boundaries, _ := json.Marshal(c.Boundaries)
	examples, _ := json.Marshal(c.Examples)
	_, err := pool.Exec(ctx, `
INSERT INTO characters (id, name, version, traits, communication_style, goals, knowledge, boundaries, examples)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
ON CONFLICT (id) DO UPDATE SET
  name=EXCLUDED.name, version=EXCLUDED.version, traits=EXCLUDED.traits,
  communication_style=EXCLUDED.communication_style, goals=EXCLUDED.goals,
  knowledge=EXCLUDED.knowledge, boundaries=EXCLUDED.boundaries, examples=EXCLUDED.examples
`, c.ID, c.Name, c.Version, traits, c.CommunicationStyle, goals, knowledge, boundaries, examples)
	return err
}

// ListAll returns all character versions ordered by name, version.
func ListAll(ctx context.Context, pool *pgxpool.Pool) ([]Character, error) {
	rows, err := pool.Query(ctx, `
SELECT id, name, version, traits, communication_style, goals, knowledge, boundaries, examples, created_at
FROM characters ORDER BY name, version`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Character
	for rows.Next() {
		var c Character
		var traits, goals, knowledge, boundaries, examples []byte
		if err := rows.Scan(&c.ID, &c.Name, &c.Version, &traits, &c.CommunicationStyle, &goals, &knowledge, &boundaries, &examples, &c.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(traits, &c.Traits)
		_ = json.Unmarshal(goals, &c.Goals)
		_ = json.Unmarshal(knowledge, &c.Knowledge)
		_ = json.Unmarshal(boundaries, &c.Boundaries)
		_ = json.Unmarshal(examples, &c.Examples)
		out = append(out, c)
	}
	return out, rows.Err()
}
