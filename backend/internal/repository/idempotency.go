package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thask/backend/internal/dbgen"
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
	q    *dbgen.Queries
}

func NewIdempotencyRepo(pool *pgxpool.Pool) *IdempotencyRepo {
	return &IdempotencyRepo{pool: pool, q: dbgen.New(pool)}
}

// Find looks up a cached response by idempotency key and API key ID.
func (r *IdempotencyRepo) Find(ctx context.Context, key, apiKeyID string) (*IdempotencyEntry, error) {
	row, err := r.q.IdempotencyFind(ctx, dbgen.IdempotencyFindParams{Key: key, ApiKeyID: apiKeyID})
	if err != nil {
		return nil, err
	}
	return idempotencyEntryFromRow(row), nil
}

// TryClaim atomically claims an idempotency key. Returns (nil, true, nil) if claimed,
// (entry, false, nil) if already exists, or (nil, false, err) on error.
func (r *IdempotencyRepo) TryClaim(ctx context.Context, key, apiKeyID, method, path string) (*IdempotencyEntry, bool, error) {
	row, err := r.q.IdempotencyTryClaim(ctx, dbgen.IdempotencyTryClaimParams{
		Key: key, ApiKeyID: apiKeyID, Method: method, Path: path,
	})
	if err != nil {
		return nil, false, err
	}
	if row.StatusCode == 0 {
		return nil, true, nil // we claimed it
	}
	return idempotencyEntryFromRow(row), false, nil // replay cached
}

// UpdateResponse updates a claimed idempotency key with the actual response.
func (r *IdempotencyRepo) UpdateResponse(ctx context.Context, key, apiKeyID string, statusCode int, response []byte) error {
	return r.q.IdempotencyUpdateResponse(ctx, dbgen.IdempotencyUpdateResponseParams{
		Key:        key,
		ApiKeyID:   apiKeyID,
		StatusCode: int32(statusCode),
		Response:   response,
	})
}

// Cleanup removes expired idempotency keys.
func (r *IdempotencyRepo) Cleanup(ctx context.Context) (int64, error) {
	return r.q.IdempotencyCleanup(ctx)
}

func idempotencyEntryFromRow(r dbgen.IdempotencyKey) *IdempotencyEntry {
	return &IdempotencyEntry{
		Key:        r.Key,
		APIKeyID:   r.ApiKeyID,
		Method:     r.Method,
		Path:       r.Path,
		StatusCode: int(r.StatusCode),
		Response:   r.Response,
		CreatedAt:  r.CreatedAt,
		ExpiresAt:  r.ExpiresAt,
	}
}
