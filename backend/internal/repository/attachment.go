package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thask/backend/internal/dbgen"
	"github.com/thask/backend/internal/model"
)

// AttachmentRepo persists file-attachment metadata for nodes (v0.6.0 B-2).
// The bytes themselves live on the filesystem under THASK_ATTACHMENT_DIR;
// storage_key is the relative path.
type AttachmentRepo struct {
	pool *pgxpool.Pool
	q    *dbgen.Queries
}

func NewAttachmentRepo(pool *pgxpool.Pool) *AttachmentRepo {
	return &AttachmentRepo{pool: pool, q: dbgen.New(pool)}
}

func attachmentFromRow(r dbgen.NodeAttachment) *model.NodeAttachment {
	return &model.NodeAttachment{
		ID:         r.ID,
		NodeID:     r.NodeID,
		ProjectID:  r.ProjectID,
		Filename:   r.Filename,
		MimeType:   r.MimeType,
		SizeBytes:  r.SizeBytes,
		StorageKey: r.StorageKey,
		SHA256:     r.Sha256,
		UploadedBy: r.UploadedBy,
		CreatedAt:  r.CreatedAt,
	}
}

func (r *AttachmentRepo) Create(ctx context.Context, nodeID, projectID, filename, mimeType string, sizeBytes int64, storageKey, sha256, uploadedBy string) (*model.NodeAttachment, error) {
	row, err := r.q.AttachmentCreate(ctx, dbgen.AttachmentCreateParams{
		NodeID:     nodeID,
		ProjectID:  projectID,
		Filename:   filename,
		MimeType:   mimeType,
		SizeBytes:  sizeBytes,
		StorageKey: storageKey,
		Sha256:     sha256,
		UploadedBy: uploadedBy,
	})
	if err != nil {
		return nil, err
	}
	return attachmentFromRow(row), nil
}

func (r *AttachmentRepo) FindByNode(ctx context.Context, nodeID string) ([]model.NodeAttachment, error) {
	rows, err := r.q.AttachmentFindByNode(ctx, nodeID)
	if err != nil {
		return nil, err
	}
	out := make([]model.NodeAttachment, len(rows))
	for i, row := range rows {
		out[i] = *attachmentFromRow(row)
	}
	return out, nil
}

func (r *AttachmentRepo) FindByID(ctx context.Context, id string) (*model.NodeAttachment, error) {
	row, err := r.q.AttachmentFindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return attachmentFromRow(row), nil
}

func (r *AttachmentRepo) Delete(ctx context.Context, id, projectID string) error {
	rows, err := r.q.AttachmentDelete(ctx, dbgen.AttachmentDeleteParams{ID: id, ProjectID: projectID})
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("attachment not found")
	}
	return nil
}
