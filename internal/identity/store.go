package identity

import (
	"context"
	"fmt"
	"strings"

	"anarchy/internal/platform"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Save upserts one identity, encrypting email/phone with key.
func Save(ctx context.Context, pool *pgxpool.Pool, key []byte, b BotIdentity) error {
	if err := b.Validate(); err != nil {
		return err
	}
	emailEnc, err := platform.EncryptString(key, b.Email)
	if err != nil {
		return err
	}
	phoneEnc, err := platform.EncryptString(key, b.Phone)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `
INSERT INTO identities (id, name, age, email_enc, phone_enc, profile, description, disabled)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
ON CONFLICT (id) DO UPDATE SET
  name=EXCLUDED.name, age=EXCLUDED.age, email_enc=EXCLUDED.email_enc, phone_enc=EXCLUDED.phone_enc,
  profile=EXCLUDED.profile, description=EXCLUDED.description, disabled=EXCLUDED.disabled
`, b.ID, b.Name, b.Age, emailEnc, phoneEnc, b.Profile, b.Description, b.Disabled)
	return err
}

// ListActive returns decrypted identities (disabled=false, deleted_at IS NULL).
// Decryption happens here only; never log raw Email/Phone.
func ListActive(ctx context.Context, pool *pgxpool.Pool, key []byte) ([]BotIdentity, error) {
	rows, err := pool.Query(ctx, `
SELECT id, name, age, email_enc, phone_enc, profile, description, disabled, created_at
FROM identities WHERE disabled=false AND deleted_at IS NULL ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []BotIdentity
	for rows.Next() {
		var b BotIdentity
		var emailEnc, phoneEnc []byte
		if err := rows.Scan(&b.ID, &b.Name, &b.Age, &emailEnc, &phoneEnc, &b.Profile, &b.Description, &b.Disabled, &b.CreatedAt); err != nil {
			return nil, err
		}
		if b.Email, err = platform.DecryptString(key, emailEnc); err != nil {
			return nil, err
		}
		if b.Phone, err = platform.DecryptString(key, phoneEnc); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// SaveAll upserts identities in chunks of 50 rows per statement, encrypting
// email/phone per row. Bulk path; a 550-row import over a remote link went
// from ~6 minutes (one query per row, timed out twice) to seconds.
func SaveAll(ctx context.Context, pool *pgxpool.Pool, key []byte, bots []BotIdentity) error {
	for _, b := range bots {
		if err := b.Validate(); err != nil {
			return err
		}
	}
	for start := 0; start < len(bots); start += 50 {
		end := start + 50
		if end > len(bots) {
			end = len(bots)
		}
		chunk := bots[start:end]
		placeholders := make([]string, 0, len(chunk))
		args := make([]any, 0, len(chunk)*8)
		for i, b := range chunk {
			emailEnc, err := platform.EncryptString(key, b.Email)
			if err != nil {
				return err
			}
			phoneEnc, err := platform.EncryptString(key, b.Phone)
			if err != nil {
				return err
			}
			base := i * 8
			placeholders = append(placeholders, fmt.Sprintf("($%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d)",
				base+1, base+2, base+3, base+4, base+5, base+6, base+7, base+8))
			args = append(args, b.ID, b.Name, b.Age, emailEnc, phoneEnc, b.Profile, b.Description, b.Disabled)
		}
		q := `INSERT INTO identities (id, name, age, email_enc, phone_enc, profile, description, disabled)
VALUES ` + strings.Join(placeholders, ",") + `
ON CONFLICT (id) DO UPDATE SET
  name=EXCLUDED.name, age=EXCLUDED.age, email_enc=EXCLUDED.email_enc, phone_enc=EXCLUDED.phone_enc,
  profile=EXCLUDED.profile, description=EXCLUDED.description, disabled=EXCLUDED.disabled`
		if _, err := pool.Exec(ctx, q, args...); err != nil {
			return err
		}
	}
	return nil
}

// SoftDelete marks identity deleted (keeps audit, drops usability).
func SoftDelete(ctx context.Context, pool *pgxpool.Pool, id string) error {
	_, err := pool.Exec(ctx, `UPDATE identities SET deleted_at=now(), disabled=true WHERE id=$1`, id)
	return err
}
