package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thask/backend/internal/model"
	"github.com/thask/backend/internal/service"
)

const projectCols = `id, team_id, name, description, created_by, link_sharing, share_token, share_token_hash, created_at, updated_at`

type ProjectRepo struct {
	pool *pgxpool.Pool
}

func NewProjectRepo(pool *pgxpool.Pool) *ProjectRepo {
	return &ProjectRepo{pool: pool}
}

func scanProject(row interface{ Scan(...any) error }) (*model.Project, error) {
	var p model.Project
	err := row.Scan(&p.ID, &p.TeamID, &p.Name, &p.Description, &p.CreatedBy, &p.LinkSharing, &p.ShareToken, &p.ShareTokenHash, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProjectRepo) Create(ctx context.Context, teamID, name string, description *string, createdBy string) (*model.Project, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO projects (team_id, name, description, created_by) VALUES ($1, $2, $3, $4)
		 RETURNING `+projectCols,
		teamID, name, description, createdBy,
	)
	return scanProject(row)
}

func (r *ProjectRepo) FindByID(ctx context.Context, id string) (*model.Project, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+projectCols+` FROM projects WHERE id = $1`, id,
	)
	return scanProject(row)
}

// GraphUpdatedAt returns the latest update time across the project's graph
// (project metadata, nodes, edges) — used for og:image cache busting.
func (r *ProjectRepo) GraphUpdatedAt(ctx context.Context, projectID string) (time.Time, error) {
	var t time.Time
	err := r.pool.QueryRow(ctx, `
		SELECT GREATEST(
			(SELECT updated_at FROM projects WHERE id = $1),
			COALESCE((SELECT MAX(updated_at) FROM nodes WHERE project_id = $1), 'epoch'::timestamptz),
			COALESCE((SELECT MAX(updated_at) FROM edges WHERE project_id = $1), 'epoch'::timestamptz)
		)`, projectID).Scan(&t)
	return t, err
}

func (r *ProjectRepo) FindByShareToken(ctx context.Context, token string) (*model.Project, error) {
	tokenHash := service.HashShareToken(token)
	row := r.pool.QueryRow(ctx,
		`SELECT `+projectCols+` FROM projects WHERE share_token_hash = $1 AND link_sharing != 'off'`, tokenHash,
	)
	return scanProject(row)
}

func (r *ProjectRepo) FindByTeamID(ctx context.Context, teamID string) ([]model.Project, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+projectCols+` FROM projects WHERE team_id = $1`, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []model.Project
	for rows.Next() {
		p, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		projects = append(projects, *p)
	}
	return projects, nil
}

func (r *ProjectRepo) FindByTeamIDs(ctx context.Context, teamIDs []string) ([]model.Project, error) {
	if len(teamIDs) == 0 {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx,
		`SELECT `+projectCols+` FROM projects WHERE team_id = ANY($1)`, teamIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []model.Project
	for rows.Next() {
		p, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		projects = append(projects, *p)
	}
	return projects, nil
}

func (r *ProjectRepo) Update(ctx context.Context, id string, name *string, description *string) (*model.Project, error) {
	row := r.pool.QueryRow(ctx,
		`UPDATE projects SET
		   name = COALESCE($1, name),
		   description = COALESCE($2, description),
		   updated_at = now()
		 WHERE id = $3
		 RETURNING `+projectCols,
		name, description, id,
	)
	return scanProject(row)
}

func (r *ProjectRepo) UpdateLinkSharing(ctx context.Context, id, linkSharing string, shareToken *string) (*model.Project, error) {
	var shareTokenHash *string
	if shareToken != nil {
		h := service.HashShareToken(*shareToken)
		shareTokenHash = &h
	}
	row := r.pool.QueryRow(ctx,
		`UPDATE projects SET
		   link_sharing = $1,
		   share_token = $2,
		   share_token_hash = $3,
		   updated_at = now()
		 WHERE id = $4
		 RETURNING `+projectCols,
		linkSharing, shareToken, shareTokenHash, id,
	)
	return scanProject(row)
}

func (r *ProjectRepo) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM projects WHERE id = $1`, id)
	return err
}

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
