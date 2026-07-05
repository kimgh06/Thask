package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thask/backend/internal/dbgen"
	"github.com/thask/backend/internal/model"
)

// ProjectTagRepo backs the canonical tag metadata table (v0.6.0 B-3). It does
// NOT own the primary tag storage — that stays on nodes.tags TEXT[]; this
// table augments those tags with color and description for autocomplete.
type ProjectTagRepo struct {
	pool *pgxpool.Pool
	q    *dbgen.Queries
}

func NewProjectTagRepo(pool *pgxpool.Pool) *ProjectTagRepo {
	return &ProjectTagRepo{pool: pool, q: dbgen.New(pool)}
}

func projectTagFromRow(r dbgen.ProjectTag) *model.ProjectTag {
	return &model.ProjectTag{
		ProjectID:   r.ProjectID,
		Tag:         r.Tag,
		Color:       r.Color,
		Description: r.Description,
		CreatedAt:   r.CreatedAt,
		CreatedBy:   r.CreatedBy,
	}
}

func (r *ProjectTagRepo) Upsert(ctx context.Context, projectID, tag string, color, description *string, createdBy string) (*model.ProjectTag, error) {
	row, err := r.q.ProjectTagUpsert(ctx, dbgen.ProjectTagUpsertParams{
		ProjectID:   projectID,
		Tag:         tag,
		Color:       color,
		Description: description,
		CreatedBy:   &createdBy,
	})
	if err != nil {
		return nil, err
	}
	return projectTagFromRow(row), nil
}

func (r *ProjectTagRepo) List(ctx context.Context, projectID string) ([]model.ProjectTag, error) {
	rows, err := r.q.ProjectTagList(ctx, projectID)
	if err != nil {
		return nil, err
	}
	out := make([]model.ProjectTag, len(rows))
	for i, row := range rows {
		out[i] = *projectTagFromRow(row)
	}
	return out, nil
}

func (r *ProjectTagRepo) Delete(ctx context.Context, projectID, tag string) error {
	rows, err := r.q.ProjectTagDelete(ctx, dbgen.ProjectTagDeleteParams{ProjectID: projectID, Tag: tag})
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("tag not found")
	}
	return nil
}
