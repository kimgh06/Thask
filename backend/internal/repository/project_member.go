package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thask/backend/internal/model"
)

type ProjectMemberRepo struct {
	pool *pgxpool.Pool
}

func NewProjectMemberRepo(pool *pgxpool.Pool) *ProjectMemberRepo {
	return &ProjectMemberRepo{pool: pool}
}

func (r *ProjectMemberRepo) GetRole(ctx context.Context, projectID, userID string) (model.ProjectRole, error) {
	var role model.ProjectRole
	err := r.pool.QueryRow(ctx,
		`SELECT role FROM project_members WHERE project_id = $1 AND user_id = $2`,
		projectID, userID,
	).Scan(&role)
	return role, err
}

func (r *ProjectMemberRepo) Add(ctx context.Context, projectID, userID string, role model.ProjectRole) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO project_members (project_id, user_id, role) VALUES ($1, $2, $3)`,
		projectID, userID, role,
	)
	return err
}

func (r *ProjectMemberRepo) UpdateRole(ctx context.Context, projectID, userID string, role model.ProjectRole) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE project_members SET role = $3 WHERE project_id = $1 AND user_id = $2`,
		projectID, userID, role,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("project member not found")
	}
	return nil
}

func (r *ProjectMemberRepo) Remove(ctx context.Context, projectID, userID string) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM project_members WHERE project_id = $1 AND user_id = $2`,
		projectID, userID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("project member not found")
	}
	return nil
}

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
