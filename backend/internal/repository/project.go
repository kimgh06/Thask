package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thask/backend/internal/model"
)

type ProjectRepo struct {
	pool *pgxpool.Pool
}

func NewProjectRepo(pool *pgxpool.Pool) *ProjectRepo {
	return &ProjectRepo{pool: pool}
}

func (r *ProjectRepo) Create(ctx context.Context, teamID, name string, description *string, createdBy string) (*model.Project, error) {
	var p model.Project
	err := r.pool.QueryRow(ctx,
		`INSERT INTO projects (team_id, name, description, created_by) VALUES ($1, $2, $3, $4)
		 RETURNING id, team_id, name, description, created_by, created_at, updated_at`,
		teamID, name, description, createdBy,
	).Scan(&p.ID, &p.TeamID, &p.Name, &p.Description, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProjectRepo) FindByID(ctx context.Context, id string) (*model.Project, error) {
	var p model.Project
	err := r.pool.QueryRow(ctx,
		`SELECT id, team_id, name, description, created_by, created_at, updated_at FROM projects WHERE id = $1`,
		id,
	).Scan(&p.ID, &p.TeamID, &p.Name, &p.Description, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProjectRepo) FindByTeamID(ctx context.Context, teamID string) ([]model.Project, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, team_id, name, description, created_by, created_at, updated_at
		 FROM projects WHERE team_id = $1`, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []model.Project
	for rows.Next() {
		var p model.Project
		if err := rows.Scan(&p.ID, &p.TeamID, &p.Name, &p.Description, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, nil
}

func (r *ProjectRepo) FindByTeamIDs(ctx context.Context, teamIDs []string) ([]model.Project, error) {
	if len(teamIDs) == 0 {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, team_id, name, description, created_by, created_at, updated_at
		 FROM projects WHERE team_id = ANY($1)`, teamIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []model.Project
	for rows.Next() {
		var p model.Project
		if err := rows.Scan(&p.ID, &p.TeamID, &p.Name, &p.Description, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, nil
}

func (r *ProjectRepo) Update(ctx context.Context, id string, name *string, description *string) (*model.Project, error) {
	var p model.Project
	err := r.pool.QueryRow(ctx,
		`UPDATE projects SET
		   name = COALESCE($1, name),
		   description = COALESCE($2, description),
		   updated_at = now()
		 WHERE id = $3
		 RETURNING id, team_id, name, description, created_by, created_at, updated_at`,
		name, description, id,
	).Scan(&p.ID, &p.TeamID, &p.Name, &p.Description, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProjectRepo) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM projects WHERE id = $1`, id)
	return err
}

// VerifyAccess checks if user has access to project via team membership
func (r *ProjectRepo) VerifyAccess(ctx context.Context, projectID, userID string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(
		   SELECT 1 FROM projects p
		   INNER JOIN team_members tm ON tm.team_id = p.team_id AND tm.user_id = $2
		   WHERE p.id = $1
		 )`, projectID, userID,
	).Scan(&exists)
	return exists, err
}
