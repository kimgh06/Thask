package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thask/backend/internal/model"
)

type APIKeyRepo struct {
	pool *pgxpool.Pool
}

func NewAPIKeyRepo(pool *pgxpool.Pool) *APIKeyRepo {
	return &APIKeyRepo{pool: pool}
}

func (r *APIKeyRepo) Create(ctx context.Context, userID, name, keyPrefix, keyHash string, expiresAt *time.Time) (*model.APIKey, error) {
	var k model.APIKey
	err := r.pool.QueryRow(ctx,
		`INSERT INTO api_keys (user_id, name, key_prefix, key_hash, expires_at)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, user_id, name, key_prefix, last_used_at, expires_at, created_at`,
		userID, name, keyPrefix, keyHash, expiresAt,
	).Scan(&k.ID, &k.UserID, &k.Name, &k.KeyPrefix, &k.LastUsedAt, &k.ExpiresAt, &k.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &k, nil
}

func (r *APIKeyRepo) FindByKeyHash(ctx context.Context, keyHash string) (*model.APIKey, *model.User, error) {
	var k model.APIKey
	var u model.User
	err := r.pool.QueryRow(ctx,
		`SELECT ak.id, ak.user_id, ak.name, ak.key_prefix, ak.last_used_at, ak.expires_at, ak.created_at,
		        u.id, u.email, u.display_name, u.created_at, u.updated_at
		 FROM api_keys ak
		 INNER JOIN users u ON ak.user_id = u.id
		 WHERE ak.key_hash = $1
		   AND (ak.expires_at IS NULL OR ak.expires_at > now())`,
		keyHash,
	).Scan(
		&k.ID, &k.UserID, &k.Name, &k.KeyPrefix, &k.LastUsedAt, &k.ExpiresAt, &k.CreatedAt,
		&u.ID, &u.Email, &u.DisplayName, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, nil, err
	}
	return &k, &u, nil
}

func (r *APIKeyRepo) FindByUserID(ctx context.Context, userID string) ([]model.APIKey, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, name, key_prefix, last_used_at, expires_at, created_at
		 FROM api_keys WHERE user_id = $1 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []model.APIKey
	for rows.Next() {
		var k model.APIKey
		if err := rows.Scan(&k.ID, &k.UserID, &k.Name, &k.KeyPrefix, &k.LastUsedAt, &k.ExpiresAt, &k.CreatedAt); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, nil
}

func (r *APIKeyRepo) Delete(ctx context.Context, id, userID string) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM api_keys WHERE id = $1 AND user_id = $2`,
		id, userID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("api key not found")
	}
	return nil
}

func (r *APIKeyRepo) UpdateLastUsed(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE api_keys SET last_used_at = now() WHERE id = $1`,
		id,
	)
	return err
}

func (r *APIKeyRepo) CountByUserID(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM api_keys WHERE user_id = $1`,
		userID,
	).Scan(&count)
	return count, err
}
