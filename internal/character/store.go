package character

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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
	quirks, _ := json.Marshal(c.Quirks)
	examples, _ := json.Marshal(c.Examples)
	_, err := pool.Exec(ctx, `
INSERT INTO characters (id, name, version, traits, communication_style, backstory, quirks, routine, goals, knowledge, boundaries, examples)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
ON CONFLICT (id) DO UPDATE SET
  name=EXCLUDED.name, version=EXCLUDED.version, traits=EXCLUDED.traits,
  communication_style=EXCLUDED.communication_style, backstory=EXCLUDED.backstory,
  quirks=EXCLUDED.quirks, routine=EXCLUDED.routine, goals=EXCLUDED.goals,
  knowledge=EXCLUDED.knowledge, boundaries=EXCLUDED.boundaries, examples=EXCLUDED.examples
`, c.ID, c.Name, c.Version, traits, c.CommunicationStyle, c.Backstory, quirks, c.Routine, goals, knowledge, boundaries, examples)
	return err
}

// SaveAll upserts characters in chunks of 50 rows per statement.
// Single-row upserts are too slow over high-latency links at bulk scale.
func SaveAll(ctx context.Context, pool *pgxpool.Pool, chars []Character) error {
	for _, c := range chars {
		if err := c.Validate(); err != nil {
			return err
		}
	}
	for start := 0; start < len(chars); start += 50 {
		end := start + 50
		if end > len(chars) {
			end = len(chars)
		}
		chunk := chars[start:end]
		placeholders := make([]string, 0, len(chunk))
		args := make([]any, 0, len(chunk)*12)
		for i, c := range chunk {
			traits, _ := json.Marshal(c.Traits)
			goals, _ := json.Marshal(c.Goals)
			knowledge, _ := json.Marshal(c.Knowledge)
			boundaries, _ := json.Marshal(c.Boundaries)
			quirks, _ := json.Marshal(c.Quirks)
			examples, _ := json.Marshal(c.Examples)
			base := i * 12
			placeholders = append(placeholders, fmt.Sprintf("($%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d)",
				base+1, base+2, base+3, base+4, base+5, base+6, base+7, base+8, base+9, base+10, base+11, base+12))
			args = append(args, c.ID, c.Name, c.Version, traits, c.CommunicationStyle, c.Backstory, quirks, c.Routine, goals, knowledge, boundaries, examples)
		}
		q := `INSERT INTO characters (id, name, version, traits, communication_style, backstory, quirks, routine, goals, knowledge, boundaries, examples)
VALUES ` + strings.Join(placeholders, ",") + `
ON CONFLICT (id) DO UPDATE SET
  name=EXCLUDED.name, version=EXCLUDED.version, traits=EXCLUDED.traits,
  communication_style=EXCLUDED.communication_style, backstory=EXCLUDED.backstory,
  quirks=EXCLUDED.quirks, routine=EXCLUDED.routine, goals=EXCLUDED.goals,
  knowledge=EXCLUDED.knowledge, boundaries=EXCLUDED.boundaries, examples=EXCLUDED.examples`
		if _, err := pool.Exec(ctx, q, args...); err != nil {
			return err
		}
	}
	return nil
}

// ListAll returns all character versions ordered by name, version.
func ListAll(ctx context.Context, pool *pgxpool.Pool) ([]Character, error) {
	rows, err := pool.Query(ctx, `
SELECT id, name, version, traits, communication_style, backstory, quirks, routine, goals, knowledge, boundaries, examples, created_at
FROM characters ORDER BY name, version`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Character
	for rows.Next() {
		var c Character
		var traits, goals, knowledge, boundaries, quirks, examples []byte
		if err := rows.Scan(&c.ID, &c.Name, &c.Version, &traits, &c.CommunicationStyle, &c.Backstory, &quirks, &c.Routine, &goals, &knowledge, &boundaries, &examples, &c.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(traits, &c.Traits)
		_ = json.Unmarshal(quirks, &c.Quirks)
		_ = json.Unmarshal(goals, &c.Goals)
		_ = json.Unmarshal(knowledge, &c.Knowledge)
		_ = json.Unmarshal(boundaries, &c.Boundaries)
		_ = json.Unmarshal(examples, &c.Examples)
		out = append(out, c)
	}
	return out, rows.Err()
}
