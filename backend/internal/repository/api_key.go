package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thask/backend/internal/dbgen"
	"github.com/thask/backend/internal/model"
)

type APIKeyRepo struct {
	pool *pgxpool.Pool
	q    *dbgen.Queries
}

func NewAPIKeyRepo(pool *pgxpool.Pool) *APIKeyRepo {
	return &APIKeyRepo{pool: pool, q: dbgen.New(pool)}
}

func apiKeyFromCreateRow(r dbgen.APIKeyCreateRow) *model.APIKey {
	return &model.APIKey{
		ID:          r.ID,
		UserID:      r.UserID,
		Name:        r.Name,
		KeyPrefix:   r.KeyPrefix,
		Kind:        model.APIKeyKind(r.Kind),
		Permissions: r.Permissions,
		LastUsedAt:  r.LastUsedAt,
		ExpiresAt:   r.ExpiresAt,
		CreatedAt:   r.CreatedAt,
	}
}

func apiKeyFromFindByUserIDRow(r dbgen.APIKeyFindByUserIDRow) model.APIKey {
	return model.APIKey{
		ID:          r.ID,
		UserID:      r.UserID,
		Name:        r.Name,
		KeyPrefix:   r.KeyPrefix,
		Kind:        model.APIKeyKind(r.Kind),
		Permissions: r.Permissions,
		LastUsedAt:  r.LastUsedAt,
		ExpiresAt:   r.ExpiresAt,
		CreatedAt:   r.CreatedAt,
	}
}

func (r *APIKeyRepo) Create(ctx context.Context, userID, name, keyPrefix, keyHash string, kind model.APIKeyKind, perms model.APIKeyPermissions, expiresAt *time.Time) (*model.APIKey, error) {
	row, err := r.q.APIKeyCreate(ctx, dbgen.APIKeyCreateParams{
		UserID:      userID,
		Name:        name,
		KeyPrefix:   keyPrefix,
		KeyHash:     keyHash,
		Kind:        string(kind),
		Permissions: perms,
		ExpiresAt:   expiresAt,
	})
	if err != nil {
		return nil, err
	}
	return apiKeyFromCreateRow(row), nil
}

// FindByKeyHash stays as raw pgx — the JOIN returns columns from two different
// tables that must be split into separate model.APIKey and model.User structs,
// which sqlc cannot express as a single mapped output type.
func (r *APIKeyRepo) FindByKeyHash(ctx context.Context, keyHash string) (*model.APIKey, *model.User, error) {
	var k model.APIKey
	var u model.User
	err := r.pool.QueryRow(ctx,
		`SELECT ak.id, ak.user_id, ak.name, ak.key_prefix, ak.kind, ak.permissions, ak.last_used_at, ak.expires_at, ak.created_at,
		        u.id, u.email, u.display_name, u.created_at, u.updated_at
		 FROM api_keys ak
		 INNER JOIN users u ON ak.user_id = u.id
		 WHERE ak.key_hash = $1
		   AND (ak.expires_at IS NULL OR ak.expires_at > now())`,
		keyHash,
	).Scan(
		&k.ID, &k.UserID, &k.Name, &k.KeyPrefix, &k.Kind, &k.Permissions, &k.LastUsedAt, &k.ExpiresAt, &k.CreatedAt,
		&u.ID, &u.Email, &u.DisplayName, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, nil, err
	}
	return &k, &u, nil
}

func (r *APIKeyRepo) FindByUserID(ctx context.Context, userID string) ([]model.APIKey, error) {
	rows, err := r.q.APIKeyFindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]model.APIKey, len(rows))
	for i, row := range rows {
		out[i] = apiKeyFromFindByUserIDRow(row)
	}
	return out, nil
}

func (r *APIKeyRepo) Delete(ctx context.Context, id, userID string) error {
	n, err := r.q.APIKeyDelete(ctx, dbgen.APIKeyDeleteParams{ID: id, UserID: userID})
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("api key not found")
	}
	return nil
}

func (r *APIKeyRepo) UpdateLastUsed(ctx context.Context, id string) error {
	return r.q.APIKeyUpdateLastUsed(ctx, id)
}

func (r *APIKeyRepo) CountByUserID(ctx context.Context, userID string) (int, error) {
	n, err := r.q.APIKeyCountByUserID(ctx, userID)
	return int(n), err
}
