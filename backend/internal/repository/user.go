package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thask/backend/internal/dbgen"
	"github.com/thask/backend/internal/model"
)

type UserRepo struct {
	pool *pgxpool.Pool
	q    *dbgen.Queries
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool, q: dbgen.New(pool)}
}

func userFromRow(r dbgen.User) *model.User {
	return &model.User{
		ID:           r.ID,
		Email:        r.Email,
		DisplayName:  r.DisplayName,
		PasswordHash: r.PasswordHash,
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
	}
}

func (r *UserRepo) Create(ctx context.Context, email, displayName, passwordHash string) (*model.User, error) {
	row, err := r.q.UserCreate(ctx, dbgen.UserCreateParams{
		Email:        email,
		DisplayName:  displayName,
		PasswordHash: passwordHash,
	})
	if err != nil {
		return nil, err
	}
	return userFromRow(row), nil
}

func (r *UserRepo) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	row, err := r.q.UserFindByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	return userFromRow(row), nil
}

func (r *UserRepo) FindByID(ctx context.Context, id string) (*model.User, error) {
	row, err := r.q.UserFindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return userFromRow(row), nil
}
