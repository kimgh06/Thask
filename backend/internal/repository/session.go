package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thask/backend/internal/model"
)

type SessionRepo struct {
	pool *pgxpool.Pool
}

func NewSessionRepo(pool *pgxpool.Pool) *SessionRepo {
	return &SessionRepo{pool: pool}
}

func (r *SessionRepo) Create(ctx context.Context, userID, token string, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO sessions (user_id, token, expires_at) VALUES ($1, $2, $3)`,
		userID, token, expiresAt,
	)
	return err
}

func (r *SessionRepo) ValidateToken(ctx context.Context, token string) (*model.User, error) {
	var u model.User
	err := r.pool.QueryRow(ctx,
		`SELECT u.id, u.email, u.display_name, u.password_hash, u.created_at, u.updated_at
		 FROM sessions s
		 INNER JOIN users u ON s.user_id = u.id
		 WHERE s.token = $1 AND s.expires_at > now()`,
		token,
	).Scan(&u.ID, &u.Email, &u.DisplayName, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *SessionRepo) DeleteByToken(ctx context.Context, token string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM sessions WHERE token = $1`, token)
	return err
}

func (r *SessionRepo) DeleteByUserID(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM sessions WHERE user_id = $1`, userID)
	return err
}

func (r *SessionRepo) DeleteExpired(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM sessions WHERE expires_at < now()`)
	return err
}
