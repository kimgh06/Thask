package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thask/backend/internal/dbgen"
	"github.com/thask/backend/internal/model"
	"github.com/thask/backend/internal/service"
)

const projectCols = `id, team_id, name, description, created_by, link_sharing, share_token, share_token_hash, created_at, updated_at`

type ProjectRepo struct {
	pool *pgxpool.Pool
	q    *dbgen.Queries
}

func NewProjectRepo(pool *pgxpool.Pool) *ProjectRepo {
	return &ProjectRepo{pool: pool, q: dbgen.New(pool)}
}

// ============================================================================
// model.Project converters (per sqlc-generated row type).
// ============================================================================

func projectFromCreateRow(r dbgen.ProjectCreateRow) *model.Project {
	return &model.Project{
		ID: r.ID, TeamID: r.TeamID, Name: r.Name, Description: r.Description,
		CreatedBy: r.CreatedBy, LinkSharing: r.LinkSharing,
		ShareToken: r.ShareToken, ShareTokenHash: r.ShareTokenHash,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func projectFromFindByIDRow(r dbgen.ProjectFindByIDRow) *model.Project {
	return &model.Project{
		ID: r.ID, TeamID: r.TeamID, Name: r.Name, Description: r.Description,
		CreatedBy: r.CreatedBy, LinkSharing: r.LinkSharing,
		ShareToken: r.ShareToken, ShareTokenHash: r.ShareTokenHash,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func projectFromFindByShareTokenRow(r dbgen.ProjectFindByShareTokenRow) *model.Project {
	return &model.Project{
		ID: r.ID, TeamID: r.TeamID, Name: r.Name, Description: r.Description,
		CreatedBy: r.CreatedBy, LinkSharing: r.LinkSharing,
		ShareToken: r.ShareToken, ShareTokenHash: r.ShareTokenHash,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func projectFromFindByTeamIDRow(r dbgen.ProjectFindByTeamIDRow) *model.Project {
	return &model.Project{
		ID: r.ID, TeamID: r.TeamID, Name: r.Name, Description: r.Description,
		CreatedBy: r.CreatedBy, LinkSharing: r.LinkSharing,
		ShareToken: r.ShareToken, ShareTokenHash: r.ShareTokenHash,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func projectFromFindByTeamIDsRow(r dbgen.ProjectFindByTeamIDsRow) *model.Project {
	return &model.Project{
		ID: r.ID, TeamID: r.TeamID, Name: r.Name, Description: r.Description,
		CreatedBy: r.CreatedBy, LinkSharing: r.LinkSharing,
		ShareToken: r.ShareToken, ShareTokenHash: r.ShareTokenHash,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func projectFromUpdateRow(r dbgen.ProjectUpdateRow) *model.Project {
	return &model.Project{
		ID: r.ID, TeamID: r.TeamID, Name: r.Name, Description: r.Description,
		CreatedBy: r.CreatedBy, LinkSharing: r.LinkSharing,
		ShareToken: r.ShareToken, ShareTokenHash: r.ShareTokenHash,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func projectFromUpdateLinkSharingRow(r dbgen.ProjectUpdateLinkSharingRow) *model.Project {
	return &model.Project{
		ID: r.ID, TeamID: r.TeamID, Name: r.Name, Description: r.Description,
		CreatedBy: r.CreatedBy, LinkSharing: r.LinkSharing,
		ShareToken: r.ShareToken, ShareTokenHash: r.ShareTokenHash,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

// scanProject is kept for the one hand-written query (GraphUpdatedAt uses raw
// pgx; Update uses raw pgx for nullable-name semantics).
func scanProject(row interface{ Scan(...any) error }) (*model.Project, error) {
	var p model.Project
	err := row.Scan(&p.ID, &p.TeamID, &p.Name, &p.Description, &p.CreatedBy, &p.LinkSharing, &p.ShareToken, &p.ShareTokenHash, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// ============================================================================
// Repo methods — static = sqlc-generated wrappers, dynamic = hand-written pgx
// ============================================================================

func (r *ProjectRepo) Create(ctx context.Context, teamID, name string, description *string, createdBy string) (*model.Project, error) {
	row, err := r.q.ProjectCreate(ctx, dbgen.ProjectCreateParams{
		TeamID:      teamID,
		Name:        name,
		Description: description,
		CreatedBy:   createdBy,
	})
	if err != nil {
		return nil, err
	}
	return projectFromCreateRow(row), nil
}

func (r *ProjectRepo) FindByID(ctx context.Context, id string) (*model.Project, error) {
	row, err := r.q.ProjectFindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return projectFromFindByIDRow(row), nil
}

// GraphUpdatedAt returns the latest update time across the project's graph
// (project metadata, nodes, edges) — used for og:image cache busting.
// Stays as hand-written pgx: multi-subquery GREATEST across 3 tables can't be
// expressed as a single sqlc query without losing type safety.
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
	row, err := r.q.ProjectFindByShareToken(ctx, &tokenHash)
	if err != nil {
		return nil, err
	}
	return projectFromFindByShareTokenRow(row), nil
}

func (r *ProjectRepo) FindByTeamID(ctx context.Context, teamID string) ([]model.Project, error) {
	rows, err := r.q.ProjectFindByTeamID(ctx, teamID)
	if err != nil {
		return nil, err
	}
	out := make([]model.Project, len(rows))
	for i, row := range rows {
		out[i] = *projectFromFindByTeamIDRow(row)
	}
	return out, nil
}

func (r *ProjectRepo) FindByTeamIDs(ctx context.Context, teamIDs []string) ([]model.Project, error) {
	if len(teamIDs) == 0 {
		return nil, nil
	}
	rows, err := r.q.ProjectFindByTeamIDs(ctx, teamIDs)
	if err != nil {
		return nil, err
	}
	out := make([]model.Project, len(rows))
	for i, row := range rows {
		out[i] = *projectFromFindByTeamIDsRow(row)
	}
	return out, nil
}

// Update stays hand-written pgx: name is nullable (*string) for partial-update
// semantics via COALESCE, but sqlc infers name as string (NOT NULL in schema),
// so the generated param can't carry a nil to mean "don't change".
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
	row, err := r.q.ProjectUpdateLinkSharing(ctx, dbgen.ProjectUpdateLinkSharingParams{
		LinkSharing:    linkSharing,
		ShareToken:     shareToken,
		ShareTokenHash: shareTokenHash,
		ID:             id,
	})
	if err != nil {
		return nil, err
	}
	return projectFromUpdateLinkSharingRow(row), nil
}

func (r *ProjectRepo) Delete(ctx context.Context, id string) error {
	return r.q.ProjectDelete(ctx, id)
}

func (r *ProjectRepo) VerifyAccess(ctx context.Context, projectID, userID string) (bool, error) {
	return r.q.ProjectVerifyAccess(ctx, dbgen.ProjectVerifyAccessParams{
		ID:     projectID,
		UserID: userID,
	})
}
