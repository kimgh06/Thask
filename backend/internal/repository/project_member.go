package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thask/backend/internal/dbgen"
	"github.com/thask/backend/internal/model"
)

type ProjectMemberRepo struct {
	pool *pgxpool.Pool
	q    *dbgen.Queries
}

func NewProjectMemberRepo(pool *pgxpool.Pool) *ProjectMemberRepo {
	return &ProjectMemberRepo{pool: pool, q: dbgen.New(pool)}
}

func (r *ProjectMemberRepo) GetRole(ctx context.Context, projectID, userID string) (model.ProjectRole, error) {
	role, err := r.q.ProjectMemberGetRole(ctx, dbgen.ProjectMemberGetRoleParams{
		ProjectID: projectID,
		UserID:    userID,
	})
	return model.ProjectRole(role), err
}

func (r *ProjectMemberRepo) Add(ctx context.Context, projectID, userID string, role model.ProjectRole) error {
	return r.q.ProjectMemberAdd(ctx, dbgen.ProjectMemberAddParams{
		ProjectID: projectID,
		UserID:    userID,
		Role:      string(role),
	})
}

func (r *ProjectMemberRepo) UpdateRole(ctx context.Context, projectID, userID string, role model.ProjectRole) error {
	rows, err := r.q.ProjectMemberUpdateRole(ctx, dbgen.ProjectMemberUpdateRoleParams{
		ProjectID: projectID,
		UserID:    userID,
		Role:      string(role),
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("project member not found")
	}
	return nil
}

func (r *ProjectMemberRepo) Remove(ctx context.Context, projectID, userID string) error {
	rows, err := r.q.ProjectMemberRemove(ctx, dbgen.ProjectMemberRemoveParams{
		ProjectID: projectID,
		UserID:    userID,
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("project member not found")
	}
	return nil
}

// ListWithUsers stays hand-written pgx: the JOIN produces a flat row that maps
// to two destination structs (ProjectMember + User) — sqlc can't split one
// row into two destination structs cleanly.
func (r *ProjectMemberRepo) ListWithUsers(ctx context.Context, projectID string) ([]model.ProjectMemberWithUser, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT pm.id, pm.project_id, pm.user_id, pm.role, pm.created_at,
		        u.id, u.email, u.display_name, u.created_at, u.updated_at
		 FROM project_members pm
		 INNER JOIN users u ON pm.user_id = u.id
		 WHERE pm.project_id = $1
		 ORDER BY pm.created_at`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []model.ProjectMemberWithUser
	for rows.Next() {
		var m model.ProjectMemberWithUser
		u := &model.User{}
		if err := rows.Scan(
			&m.ID, &m.ProjectID, &m.UserID, &m.Role, &m.CreatedAt,
			&u.ID, &u.Email, &u.DisplayName, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, err
		}
		m.User = u
		members = append(members, m)
	}
	return members, nil
}
