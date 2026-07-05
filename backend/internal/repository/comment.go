package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thask/backend/internal/dbgen"
	"github.com/thask/backend/internal/model"
)

// CommentRepo wraps sqlc-generated queries against node_comments (v0.6.0 B-1).
type CommentRepo struct {
	pool *pgxpool.Pool
	q    *dbgen.Queries
}

func NewCommentRepo(pool *pgxpool.Pool) *CommentRepo {
	return &CommentRepo{pool: pool, q: dbgen.New(pool)}
}

func commentFromRow(r dbgen.NodeComment) *model.NodeComment {
	return &model.NodeComment{
		ID:         r.ID,
		NodeID:     r.NodeID,
		ProjectID:  r.ProjectID,
		AuthorID:   r.AuthorID,
		ParentID:   r.ParentID,
		Body:       r.Body,
		ResolvedAt: r.ResolvedAt,
		ResolvedBy: r.ResolvedBy,
		CreatedAt:  r.CreatedAt,
		UpdatedAt:  r.UpdatedAt,
	}
}

func (r *CommentRepo) Create(ctx context.Context, nodeID, projectID, authorID string, parentID *string, body string) (*model.NodeComment, error) {
	row, err := r.q.CommentCreate(ctx, dbgen.CommentCreateParams{
		NodeID:    nodeID,
		ProjectID: projectID,
		AuthorID:  authorID,
		ParentID:  parentID,
		Body:      body,
	})
	if err != nil {
		return nil, err
	}
	return commentFromRow(row), nil
}

func (r *CommentRepo) FindByNode(ctx context.Context, nodeID string) ([]model.NodeComment, error) {
	rows, err := r.q.CommentFindByNode(ctx, nodeID)
	if err != nil {
		return nil, err
	}
	out := make([]model.NodeComment, len(rows))
	for i, row := range rows {
		out[i] = *commentFromRow(row)
	}
	return out, nil
}

func (r *CommentRepo) FindProjectUnresolved(ctx context.Context, projectID string, limit int) ([]model.NodeComment, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.q.CommentFindByProjectUnresolved(ctx, dbgen.CommentFindByProjectUnresolvedParams{
		ProjectID: projectID,
		Limit:     int32(limit),
	})
	if err != nil {
		return nil, err
	}
	out := make([]model.NodeComment, len(rows))
	for i, row := range rows {
		out[i] = *commentFromRow(row)
	}
	return out, nil
}

func (r *CommentRepo) UpdateBody(ctx context.Context, id, projectID, authorID, body string) (*model.NodeComment, error) {
	row, err := r.q.CommentUpdateBody(ctx, dbgen.CommentUpdateBodyParams{
		Body:      body,
		ID:        id,
		ProjectID: projectID,
		AuthorID:  authorID,
	})
	if err != nil {
		return nil, err
	}
	return commentFromRow(row), nil
}

func (r *CommentRepo) Resolve(ctx context.Context, id, projectID, resolvedBy string) (*model.NodeComment, error) {
	row, err := r.q.CommentResolve(ctx, dbgen.CommentResolveParams{
		ResolvedBy: &resolvedBy,
		ID:         id,
		ProjectID:  projectID,
	})
	if err != nil {
		return nil, err
	}
	return commentFromRow(row), nil
}

func (r *CommentRepo) Delete(ctx context.Context, id, projectID, authorID string) error {
	rows, err := r.q.CommentDelete(ctx, dbgen.CommentDeleteParams{ID: id, ProjectID: projectID, AuthorID: authorID})
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("comment not found or not owned by caller")
	}
	return nil
}
