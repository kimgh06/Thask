package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thask/backend/internal/dbgen"
	"github.com/thask/backend/internal/model"
)

type SessionRepo struct {
	pool *pgxpool.Pool
	q    *dbgen.Queries
}

func NewSessionRepo(pool *pgxpool.Pool) *SessionRepo {
	return &SessionRepo{pool: pool, q: dbgen.New(pool)}
}

func (r *SessionRepo) Create(ctx context.Context, userID, token string, expiresAt time.Time) error {
	return r.q.SessionCreate(ctx, dbgen.SessionCreateParams{
		UserID:    userID,
		Token:     token,
		ExpiresAt: expiresAt,
	})
}

func (r *SessionRepo) ValidateToken(ctx context.Context, token string) (*model.User, error) {
	row, err := r.q.SessionValidateToken(ctx, token)
	if err != nil {
		return nil, err
	}
	return userFromRow(row), nil
}

func (r *SessionRepo) DeleteByToken(ctx context.Context, token string) error {
	return r.q.SessionDeleteByToken(ctx, token)
}

func (r *SessionRepo) DeleteByUserID(ctx context.Context, userID string) error {
	return r.q.SessionDeleteByUserID(ctx, userID)
}

func (r *SessionRepo) DeleteExpired(ctx context.Context) error {
	return r.q.SessionDeleteExpired(ctx)
}
