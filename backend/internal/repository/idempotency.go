package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// IdempotencyEntry represents a cached response for an idempotent request.
type IdempotencyEntry struct {
	Key        string    `db:"key"`
	APIKeyID   string    `db:"api_key_id"`
	Method     string    `db:"method"`
	Path       string    `db:"path"`
	StatusCode int       `db:"status_code"`
	Response   []byte    `db:"response"`
	CreatedAt  time.Time `db:"created_at"`
	ExpiresAt  time.Time `db:"expires_at"`
}

type IdempotencyRepo struct {
	pool *pgxpool.Pool
}

func NewIdempotencyRepo(pool *pgxpool.Pool) *IdempotencyRepo {
	return &IdempotencyRepo{pool: pool}
}

// Find looks up a cached response by idempotency key and API key ID.
func (r *IdempotencyRepo) Find(ctx context.Context, key, apiKeyID string) (*IdempotencyEntry, error) {
	var e IdempotencyEntry
	err := r.pool.QueryRow(ctx,
		`SELECT key, api_key_id, method, path, status_code, response, created_at, expires_at
		 FROM idempotency_keys
		 WHERE key = $1 AND api_key_id = $2 AND expires_at > now()`,
		key, apiKeyID,
	).Scan(&e.Key, &e.APIKeyID, &e.Method, &e.Path, &e.StatusCode, &e.Response, &e.CreatedAt, &e.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// TryClaim atomically claims an idempotency key. Returns (nil, true, nil) if claimed,
// (entry, false, nil) if already exists, or (nil, false, err) on error.
func (r *IdempotencyRepo) TryClaim(ctx context.Context, key, apiKeyID, method, path string) (*IdempotencyEntry, bool, error) {
	var e IdempotencyEntry
	err := r.pool.QueryRow(ctx,
		`INSERT INTO idempotency_keys (key, api_key_id, method, path, status_code, response)
		 VALUES ($1, $2, $3, $4, 0, '{}'::jsonb)
		 ON CONFLICT (key, api_key_id) DO UPDATE SET key = idempotency_keys.key
		 RETURNING key, api_key_id, method, path, status_code, response, created_at, expires_at`,
		key, apiKeyID, method, path,
	).Scan(&e.Key, &e.APIKeyID, &e.Method, &e.Path, &e.StatusCode, &e.Response, &e.CreatedAt, &e.ExpiresAt)
	if err != nil {
		return nil, false, err
	}
	if e.StatusCode == 0 {
		return nil, true, nil // we claimed it
	}
	return &e, false, nil // replay cached
}

// UpdateResponse updates a claimed idempotency key with the actual response.
func (r *IdempotencyRepo) UpdateResponse(ctx context.Context, key, apiKeyID string, statusCode int, response []byte) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE idempotency_keys SET status_code = $3, response = $4 WHERE key = $1 AND api_key_id = $2`,
		key, apiKeyID, statusCode, response,
	)
	return err
}

// Cleanup removes expired idempotency keys.
func (r *IdempotencyRepo) Cleanup(ctx context.Context) (int64, error) {
	tag, err := r.pool.Exec(ctx, `DELETE FROM idempotency_keys WHERE expires_at < now()`)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
