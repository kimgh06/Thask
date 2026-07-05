package handler

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/thask/backend/internal/audit"
	"github.com/thask/backend/internal/dto"
	mw "github.com/thask/backend/internal/middleware"
	"github.com/thask/backend/internal/repository"
)

// KnowledgeOSHandler bundles the v0.6.0 side-tables: comments, attachments,
// and project tags. Kept in one struct so routes.go only wires one handler.
type KnowledgeOSHandler struct {
	nodeRepo       *repository.NodeRepo
	commentRepo    *repository.CommentRepo
	attachmentRepo *repository.AttachmentRepo
	tagRepo        *repository.ProjectTagRepo
	audit          *audit.Logger

	attachmentDir      string
	attachmentMaxBytes int64
}

func NewKnowledgeOSHandler(
	nodeRepo *repository.NodeRepo,
	commentRepo *repository.CommentRepo,
	attachmentRepo *repository.AttachmentRepo,
	tagRepo *repository.ProjectTagRepo,
	auditLogger *audit.Logger,
	attachmentDir string,
	attachmentMaxBytes int64,
) *KnowledgeOSHandler {
	return &KnowledgeOSHandler{
		nodeRepo:           nodeRepo,
		commentRepo:        commentRepo,
		attachmentRepo:     attachmentRepo,
		tagRepo:            tagRepo,
		audit:              auditLogger,
		attachmentDir:      attachmentDir,
		attachmentMaxBytes: attachmentMaxBytes,
	}
}

// verifyNodeInProject rejects requests whose :nodeId URL param does not
// belong to the project the caller has access to. Without this check, a
// caller with access to project A can insert / list comments and
// attachments on nodes in project B by supplying a valid but unrelated
// node UUID. Returns nil on success, a JSON 404 error otherwise
// (404 not 403 — hides node existence from cross-project probes).
func (h *KnowledgeOSHandler) verifyNodeInProject(c echo.Context, projectID, nodeID string) error {
	if _, err := h.nodeRepo.FindByID(c.Request().Context(), nodeID, projectID); err != nil {
		return c.JSON(http.StatusNotFound, dto.Err("Node not found"))
	}
	return nil
}

// ─── Comments ────────────────────────────────────────────────────────────────

func (h *KnowledgeOSHandler) ListComments(c echo.Context) error {
	projectID := mw.ResolveProjectID(c)
	nodeID := c.Param("nodeId")
	if err := h.verifyNodeInProject(c, projectID, nodeID); err != nil {
		return err
	}
	rows, err := h.commentRepo.FindByNode(c.Request().Context(), nodeID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to list comments"))
	}
	return c.JSON(http.StatusOK, dto.OK(rows))
}

func (h *KnowledgeOSHandler) CreateComment(c echo.Context) error {
	var req dto.CommentCreateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err("Invalid request body"))
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err(err.Error()))
	}
	ctx := c.Request().Context()
	projectID := mw.ResolveProjectID(c)
	nodeID := c.Param("nodeId")
	userID := mw.GetUserID(c)

	if err := h.verifyNodeInProject(c, projectID, nodeID); err != nil {
		return err
	}
	if err := h.audit.RequirePermission(c, audit.MutationMeta, "write", audit.Event{
		ProjectID: projectID, EntityType: "node", EntityID: nodeID, Action: audit.ActionUpdated,
	}); err != nil {
		return err
	}

	created, err := h.commentRepo.Create(ctx, nodeID, projectID, userID, req.ParentID, req.Body)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to create comment"))
	}
	h.audit.Log(c, audit.Event{
		ProjectID: projectID, EntityType: "node", EntityID: nodeID,
		Action: audit.ActionUpdated, MutationKind: audit.MutationMeta,
		FieldName: "comment", NewValue: created.ID,
	})
	return c.JSON(http.StatusCreated, dto.OK(created))
}

func (h *KnowledgeOSHandler) UpdateComment(c echo.Context) error {
	var req dto.CommentUpdateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err("Invalid request body"))
	}
	if req.Body == nil {
		return c.JSON(http.StatusBadRequest, dto.Err("Nothing to update"))
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err(err.Error()))
	}
	projectID := mw.ResolveProjectID(c)
	commentID := c.Param("commentId")
	userID := mw.GetUserID(c)
	if err := h.audit.RequirePermission(c, audit.MutationMeta, "write", audit.Event{
		ProjectID: projectID, EntityType: "node", Action: audit.ActionUpdated, FieldName: "comment",
	}); err != nil {
		return err
	}
	updated, err := h.commentRepo.UpdateBody(c.Request().Context(), commentID, projectID, userID, *req.Body)
	if err != nil {
		return c.JSON(http.StatusNotFound, dto.Err("Comment not found or not owned by caller"))
	}
	h.audit.Log(c, audit.Event{
		ProjectID: projectID, EntityType: "node", EntityID: updated.NodeID,
		Action: audit.ActionUpdated, MutationKind: audit.MutationMeta,
		FieldName: "comment", NewValue: commentID,
	})
	return c.JSON(http.StatusOK, dto.OK(updated))
}

func (h *KnowledgeOSHandler) ResolveComment(c echo.Context) error {
	projectID := mw.ResolveProjectID(c)
	commentID := c.Param("commentId")
	userID := mw.GetUserID(c)
	if err := h.audit.RequirePermission(c, audit.MutationMeta, "write", audit.Event{
		ProjectID: projectID, EntityType: "node", Action: audit.ActionUpdated, FieldName: "comment_resolved",
	}); err != nil {
		return err
	}
	updated, err := h.commentRepo.Resolve(c.Request().Context(), commentID, projectID, userID)
	if err != nil {
		return c.JSON(http.StatusNotFound, dto.Err("Comment not found or already resolved"))
	}
	h.audit.Log(c, audit.Event{
		ProjectID: projectID, EntityType: "node", EntityID: updated.NodeID,
		Action: audit.ActionUpdated, MutationKind: audit.MutationMeta,
		FieldName: "comment_resolved", NewValue: commentID,
	})
	return c.JSON(http.StatusOK, dto.OK(updated))
}

func (h *KnowledgeOSHandler) DeleteComment(c echo.Context) error {
	projectID := mw.ResolveProjectID(c)
	commentID := c.Param("commentId")
	userID := mw.GetUserID(c)
	if err := h.audit.RequirePermission(c, audit.MutationMeta, "delete", audit.Event{
		ProjectID: projectID, EntityType: "node", Action: audit.ActionDeleted, FieldName: "comment",
	}); err != nil {
		return err
	}
	if err := h.commentRepo.Delete(c.Request().Context(), commentID, projectID, userID); err != nil {
		return c.JSON(http.StatusNotFound, dto.Err(err.Error()))
	}
	h.audit.Log(c, audit.Event{
		ProjectID: projectID, EntityType: "node",
		Action: audit.ActionDeleted, MutationKind: audit.MutationMeta,
		FieldName: "comment", OldValue: commentID,
	})
	return c.JSON(http.StatusOK, dto.OK(dto.SuccessResponse{Success: true}))
}

// ─── Attachments ─────────────────────────────────────────────────────────────

func (h *KnowledgeOSHandler) ListAttachments(c echo.Context) error {
	projectID := mw.ResolveProjectID(c)
	nodeID := c.Param("nodeId")
	if err := h.verifyNodeInProject(c, projectID, nodeID); err != nil {
		return err
	}
	rows, err := h.attachmentRepo.FindByNode(c.Request().Context(), nodeID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to list attachments"))
	}
	return c.JSON(http.StatusOK, dto.OK(rows))
}

func (h *KnowledgeOSHandler) UploadAttachment(c echo.Context) error {
	if h.attachmentDir == "" {
		return c.JSON(http.StatusServiceUnavailable, dto.Err("Attachments disabled: THASK_ATTACHMENT_DIR unset"))
	}
	ctx := c.Request().Context()
	projectID := mw.ResolveProjectID(c)
	nodeID := c.Param("nodeId")
	userID := mw.GetUserID(c)

	if err := h.verifyNodeInProject(c, projectID, nodeID); err != nil {
		return err
	}
	if err := h.audit.RequirePermission(c, audit.MutationMeta, "write", audit.Event{
		ProjectID: projectID, EntityType: "node", EntityID: nodeID, Action: audit.ActionUpdated,
	}); err != nil {
		return err
	}

	// Cap request body BEFORE Echo parses multipart, so an attacker can't spool
	// 100GB to /tmp waiting for our post-parse size check to run.
	c.Request().Body = http.MaxBytesReader(c.Response().Writer, c.Request().Body, h.attachmentMaxBytes)

	file, err := c.FormFile("file")
	if err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err("file field required"))
	}
	if file.Size > h.attachmentMaxBytes {
		return c.JSON(http.StatusRequestEntityTooLarge, dto.Err(fmt.Sprintf("file exceeds max size %d", h.attachmentMaxBytes)))
	}
	src, err := file.Open()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to open upload"))
	}
	defer src.Close()

	// storage_key layout: {projectId}/{yyyy}/{uuid}-{safeFilename}. Sharding by
	// project keeps cascade-delete a single-directory concern later.
	safeName := filepath.Base(file.Filename)
	safeName = strings.ReplaceAll(safeName, "..", "_")
	// 16 bytes random hex is enough entropy for storage_key uniqueness
	// without pulling in a UUID dep just for this one path.
	var rnd [16]byte
	if _, err := rand.Read(rnd[:]); err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to generate storage key"))
	}
	storageKey := filepath.Join(projectID, hex.EncodeToString(rnd[:])+"-"+safeName)
	fullPath := filepath.Join(h.attachmentDir, storageKey)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to prepare storage dir"))
	}
	dst, err := os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to open destination file"))
	}
	hash := sha256.New()
	n, err := io.Copy(io.MultiWriter(dst, hash), src)
	closeErr := dst.Close()
	if err != nil || closeErr != nil {
		_ = os.Remove(fullPath) // don't leak partials
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to persist file"))
	}
	sha := hex.EncodeToString(hash.Sum(nil))

	mimeType := file.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	created, err := h.attachmentRepo.Create(ctx, nodeID, projectID, safeName, mimeType, n, storageKey, sha, userID)
	if err != nil {
		_ = os.Remove(fullPath) // roll back the file since DB row failed
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to record attachment"))
	}
	h.audit.Log(c, audit.Event{
		ProjectID: projectID, EntityType: "node", EntityID: nodeID,
		Action: audit.ActionUpdated, MutationKind: audit.MutationMeta,
		FieldName: "attachment", NewValue: created.ID,
	})
	return c.JSON(http.StatusCreated, dto.OK(created))
}

func (h *KnowledgeOSHandler) DownloadAttachment(c echo.Context) error {
	if h.attachmentDir == "" {
		return c.JSON(http.StatusServiceUnavailable, dto.Err("Attachments disabled"))
	}
	projectID := mw.ResolveProjectID(c)
	att, err := h.attachmentRepo.FindByID(c.Request().Context(), c.Param("attachmentId"))
	if err != nil {
		return c.JSON(http.StatusNotFound, dto.Err("Attachment not found"))
	}
	// IDOR guard (v0.6.0 review CVE-KOS-001): FindByID is global — the row's
	// project_id MUST match the URL's project scope. Otherwise a caller with
	// access to project A can download files from project B by guessing UUIDs.
	// 404 (not 403) to avoid leaking that the attachment exists elsewhere.
	if att.ProjectID != projectID {
		return c.JSON(http.StatusNotFound, dto.Err("Attachment not found"))
	}
	// Belt-and-suspenders: ensure the stored path stays inside attachmentDir.
	// Rejects path-traversal even if a bad storage_key ever landed in the DB.
	full := filepath.Join(h.attachmentDir, att.StorageKey)
	rel, err := filepath.Rel(h.attachmentDir, full)
	if err != nil || strings.HasPrefix(rel, "..") {
		return c.JSON(http.StatusForbidden, dto.Err("Invalid storage key"))
	}
	// Force download + block MIME sniffing so an attacker can't upload a file
	// that browsers render as HTML/SVG/JS (XSS on the Thask session). CSP
	// sandbox contains any content that would otherwise execute (Flash / PDFs
	// that embed scripts / etc.).
	c.Response().Header().Set("Content-Disposition", `attachment; filename="`+sanitizeDownloadFilename(att.Filename)+`"`)
	c.Response().Header().Set("X-Content-Type-Options", "nosniff")
	c.Response().Header().Set("Content-Security-Policy", "sandbox")
	return c.File(full)
}

// sanitizeDownloadFilename strips characters that would break the RFC 6266
// Content-Disposition header or let an attacker inject header lines.
func sanitizeDownloadFilename(name string) string {
	name = strings.Map(func(r rune) rune {
		switch r {
		case '"', '\\', '\r', '\n':
			return '_'
		}
		return r
	}, name)
	if name == "" {
		return "download"
	}
	return name
}

func (h *KnowledgeOSHandler) DeleteAttachment(c echo.Context) error {
	ctx := c.Request().Context()
	projectID := mw.ResolveProjectID(c)
	id := c.Param("attachmentId")

	att, err := h.attachmentRepo.FindByID(ctx, id)
	if err != nil {
		return c.JSON(http.StatusNotFound, dto.Err("Attachment not found"))
	}
	// IDOR guard: same rule as DownloadAttachment. Delete-scoped by projectID
	// in the query would also work, but the FindByID here also gives us the
	// storage_key we need to unlink the blob after DB delete succeeds.
	if att.ProjectID != projectID {
		return c.JSON(http.StatusNotFound, dto.Err("Attachment not found"))
	}
	if err := h.audit.RequirePermission(c, audit.MutationMeta, "delete", audit.Event{
		ProjectID: projectID, EntityType: "node", EntityID: att.NodeID,
		Action: audit.ActionDeleted, FieldName: "attachment",
	}); err != nil {
		return err
	}
	if err := h.attachmentRepo.Delete(ctx, id, projectID); err != nil {
		return c.JSON(http.StatusNotFound, dto.Err(err.Error()))
	}
	// Log the blob-remove failure so ops can garbage-collect strays later —
	// row is already gone, so we prefer a leaked byte over a dangling row.
	if h.attachmentDir != "" {
		if rmErr := os.Remove(filepath.Join(h.attachmentDir, att.StorageKey)); rmErr != nil && !os.IsNotExist(rmErr) {
			slog.Warn("attachment blob leak (DB row removed, file not)",
				"storage_key", att.StorageKey, "error", rmErr)
		}
	}
	h.audit.Log(c, audit.Event{
		ProjectID: projectID, EntityType: "node", EntityID: att.NodeID,
		Action: audit.ActionDeleted, MutationKind: audit.MutationMeta,
		FieldName: "attachment", OldValue: id,
	})
	return c.JSON(http.StatusOK, dto.OK(dto.SuccessResponse{Success: true}))
}

// ─── Project tags ────────────────────────────────────────────────────────────

func (h *KnowledgeOSHandler) ListTags(c echo.Context) error {
	rows, err := h.tagRepo.List(c.Request().Context(), mw.ResolveProjectID(c))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to list tags"))
	}
	return c.JSON(http.StatusOK, dto.OK(rows))
}

func (h *KnowledgeOSHandler) UpsertTag(c echo.Context) error {
	tag := c.Param("tag")
	if tag == "" {
		return c.JSON(http.StatusBadRequest, dto.Err("tag path param required"))
	}
	var req dto.ProjectTagUpsertRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err("Invalid request body"))
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err(err.Error()))
	}
	projectID := mw.ResolveProjectID(c)
	userID := mw.GetUserID(c)
	if err := h.audit.RequirePermission(c, audit.MutationMeta, "write", audit.Event{
		ProjectID: projectID, EntityType: "project", EntityID: projectID,
		Action: audit.ActionUpdated, FieldName: "tag",
	}); err != nil {
		return err
	}
	created, err := h.tagRepo.Upsert(c.Request().Context(), projectID, tag, req.Color, req.Description, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to upsert tag"))
	}
	h.audit.Log(c, audit.Event{
		ProjectID: projectID, EntityType: "project", EntityID: projectID,
		Action: audit.ActionUpdated, MutationKind: audit.MutationMeta,
		FieldName: "tag", NewValue: tag,
	})
	return c.JSON(http.StatusOK, dto.OK(created))
}

func (h *KnowledgeOSHandler) DeleteTag(c echo.Context) error {
	projectID := mw.ResolveProjectID(c)
	tag := c.Param("tag")
	if err := h.audit.RequirePermission(c, audit.MutationMeta, "delete", audit.Event{
		ProjectID: projectID, EntityType: "project", EntityID: projectID,
		Action: audit.ActionDeleted, FieldName: "tag",
	}); err != nil {
		return err
	}
	if err := h.tagRepo.Delete(c.Request().Context(), projectID, tag); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return c.JSON(http.StatusNotFound, dto.Err(err.Error()))
		}
		return c.JSON(http.StatusNotFound, dto.Err(err.Error()))
	}
	h.audit.Log(c, audit.Event{
		ProjectID: projectID, EntityType: "project", EntityID: projectID,
		Action: audit.ActionDeleted, MutationKind: audit.MutationMeta,
		FieldName: "tag", OldValue: tag,
	})
	return c.JSON(http.StatusOK, dto.OK(dto.SuccessResponse{Success: true}))
}
