// Package audit centralises provenance-tagged write logging and per-key
// permission gating. It lives outside `service` to avoid an import cycle
// (service ↔ repository ↔ middleware), and is imported directly by handlers.
package audit

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/thask/backend/internal/dto"
	mw "github.com/thask/backend/internal/middleware"
	"github.com/thask/backend/internal/model"
	"github.com/thask/backend/internal/repository"
)

// Mutation kinds — keep in sync with audit_log CHECK constraint (migration 006)
// and APIKeyPermissions.Allows in model/models.go.
const (
	MutationStructural = "structural"
	MutationSemantic   = "semantic"
	MutationMeta       = "meta"
)

// Audit actions — strings used in audit_log.action.
const (
	ActionCreated        = "created"
	ActionUpdated        = "updated"
	ActionDeleted        = "deleted"
	ActionImported       = "imported"
	ActionScanned        = "scanned"
	ActionVerified       = "verified"
	ActionSuggested      = "suggested"
	ActionSuggestDecided = "suggestion_decided"
	ActionWriteDenied    = "write_denied"
	ActionStatusChanged  = "status_changed"
	ActionLayoutComputed = "layout_computed"
)

// Logger writes provenance-tagged events to audit_log. Failures are best-effort:
// a logging error never blocks the originating handler, because losing one
// audit row is strictly preferable to losing the user's write.
type Logger struct {
	repo *repository.AuditRepo
}

func NewLogger(repo *repository.AuditRepo) *Logger {
	return &Logger{repo: repo}
}

// Event is the per-call payload. ProjectID is required. All other text fields
// are optional ("" means "omit from row").
type Event struct {
	ProjectID    string
	EntityType   string // "node" | "edge" | "project" | "graph" | "suggestion"
	EntityID     string
	Action       string
	FieldName    string
	OldValue     string
	NewValue     string
	MutationKind string // defaults to MutationMeta
	Trigger      string
	BatchID      string
	CodeCommit   string
	SourcePath   string
	Confidence   string
	Metadata     any
}

// Log records the event, attributing it to whoever the middleware identified.
// Errors are logged but not returned — the caller's primary work should not
// fail because the audit trail couldn't be written.
func (l *Logger) Log(c echo.Context, evt Event) {
	if l == nil || l.repo == nil {
		return
	}
	if evt.MutationKind == "" {
		evt.MutationKind = MutationMeta
	}

	row := &model.AuditLog{
		ProjectID:      evt.ProjectID,
		EntityType:     evt.EntityType,
		Action:         evt.Action,
		MutationKind:   evt.MutationKind,
		ActorKind:      mw.GetActorKind(c),
		EntityID:       strPtr(evt.EntityID),
		FieldName:      strPtr(evt.FieldName),
		OldValue:       strPtr(evt.OldValue),
		NewValue:       strPtr(evt.NewValue),
		Trigger:        strPtr(evt.Trigger),
		BatchID:        strPtr(evt.BatchID),
		CodeCommit:     strPtr(evt.CodeCommit),
		SourcePath:     strPtr(evt.SourcePath),
		Confidence:     strPtr(evt.Confidence),
		ClientType:     strPtr(mw.GetClientType(c)),
		ClientVersion:  strPtr(mw.GetClientVersion(c)),
		AgentModel:     strPtr(mw.GetAgentModel(c)),
		AgentSessionID: strPtr(mw.GetAgentSessionID(c)),
		Metadata:       evt.Metadata,
	}
	if uid := safeUserID(c); uid != "" {
		row.UserID = &uid
	}
	if akid := mw.GetAPIKeyID(c); akid != "" {
		row.APIKeyID = &akid
	}

	// Use a detached context so the audit row still lands even if the request
	// context was cancelled (e.g. client disconnect after we already mutated).
	if err := l.repo.Insert(context.Background(), row); err != nil {
		slog.Warn("audit_log insert failed",
			"entity_type", evt.EntityType,
			"action", evt.Action,
			"project_id", evt.ProjectID,
			"error", err)
	}
}

// RequirePermission gates a mutation by the caller's per-key permissions.
// Returns an echo HTTP error (403) on denial; the caller should simply return
// it. The denial is itself recorded to audit_log so users can see "this key
// would have written X but lacked permission Y".
func (l *Logger) RequirePermission(c echo.Context, mutationKind, action string, evt Event) error {
	perms := mw.GetPermissions(c)
	if perms.Allows(mutationKind, action) {
		return nil
	}

	required := permissionFlagFor(mutationKind, action)
	denyEvt := evt
	denyEvt.Action = ActionWriteDenied
	denyEvt.MutationKind = mutationKind
	denyEvt.Metadata = map[string]any{
		"required_permission": required,
		"attempted_action":    action,
		"actor_kind":          mw.GetActorKind(c),
	}
	l.Log(c, denyEvt)

	return c.JSON(http.StatusForbidden, dto.Err(
		"This API key lacks the '"+required+"' permission. "+
			suggestionHint(mutationKind, action),
	))
}

func permissionFlagFor(mutationKind, action string) string {
	switch action {
	case "read":
		return "read"
	case "delete":
		return "delete"
	case "verify":
		return "verify"
	case "suggest":
		return "suggest"
	case "write":
		switch mutationKind {
		case MutationStructural:
			return "write_structural"
		case MutationSemantic:
			return "write_semantic"
		case MutationMeta:
			return "write_meta"
		}
	}
	return action
}

func suggestionHint(mutationKind, action string) string {
	switch {
	case action == "write" && mutationKind == MutationSemantic:
		return "Use thask.node.suggest_update to propose this change for human review."
	case action == "verify":
		return "Only user_interactive keys (or keys with verify=true) can mark a node as verified."
	}
	return "Contact the key owner to grant the permission."
}

// safeUserID recovers from a nil context user ID (e.g. anonymous shared access
// before middleware sets it) so we never panic on the audit path.
func safeUserID(c echo.Context) string {
	defer func() { _ = recover() }()
	v, _ := c.Get(mw.ContextUserID).(string)
	return v
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
