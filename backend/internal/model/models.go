package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

type User struct {
	ID           string    `json:"id" db:"id"`
	Email        string    `json:"email" db:"email"`
	DisplayName  string    `json:"displayName" db:"display_name"`
	PasswordHash string    `json:"-" db:"password_hash"`
	CreatedAt    time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt    time.Time `json:"updatedAt" db:"updated_at"`
}

type Session struct {
	ID        string    `json:"id" db:"id"`
	UserID    string    `json:"userId" db:"user_id"`
	Token     string    `json:"token" db:"token"`
	ExpiresAt time.Time `json:"expiresAt" db:"expires_at"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
}

type Team struct {
	ID        string    `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Slug      string    `json:"slug" db:"slug"`
	CreatedBy string    `json:"createdBy" db:"created_by"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

type TeamWithProjects struct {
	Team
	Projects []Project `json:"projects"`
}

type TeamMember struct {
	ID       string   `json:"id" db:"id"`
	TeamID   string   `json:"teamId" db:"team_id"`
	UserID   string   `json:"userId" db:"user_id"`
	Role     TeamRole `json:"role" db:"role"`
	JoinedAt time.Time `json:"joinedAt" db:"joined_at"`
}

type TeamMemberWithUser struct {
	TeamMember
	User *User `json:"user,omitempty"`
}

type Project struct {
	ID          string    `json:"id" db:"id"`
	TeamID      string    `json:"teamId" db:"team_id"`
	Name        string    `json:"name" db:"name"`
	Description *string   `json:"description" db:"description"`
	CreatedBy   string    `json:"createdBy" db:"created_by"`
	LinkSharing    string    `json:"linkSharing" db:"link_sharing"`
	ShareToken     *string   `json:"-" db:"share_token"`
	ShareTokenHash *string   `json:"-" db:"share_token_hash"`
	CreatedAt      time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt      time.Time `json:"updatedAt" db:"updated_at"`
}

type ProjectMember struct {
	ID        string      `json:"id" db:"id"`
	ProjectID string      `json:"projectId" db:"project_id"`
	UserID    string      `json:"userId" db:"user_id"`
	Role      ProjectRole `json:"role" db:"role"`
	CreatedAt time.Time   `json:"createdAt" db:"created_at"`
}

type ProjectMemberWithUser struct {
	ProjectMember
	User *User `json:"user,omitempty"`
}

type Node struct {
	ID          string     `json:"id" db:"id"`
	ProjectID   string     `json:"projectId" db:"project_id"`
	Type        NodeType   `json:"type" db:"type"`
	Title       string     `json:"title" db:"title"`
	Description *string    `json:"description" db:"description"`
	Status      NodeStatus `json:"status" db:"status"`
	AssigneeID  *string    `json:"assigneeId" db:"assignee_id"`
	Tags        []string   `json:"tags" db:"tags"`
	Metadata    any        `json:"metadata" db:"metadata"`
	ParentID    *string    `json:"parentId" db:"parent_id"`
	PositionX   float64    `json:"positionX" db:"position_x"`
	PositionY   float64    `json:"positionY" db:"position_y"`
	Width       *float64   `json:"width" db:"width"`
	Height      *float64   `json:"height" db:"height"`
	CreatedAt   time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time  `json:"updatedAt" db:"updated_at"`

	// Provenance (migration 007). Describes who/what produced the current
	// description, and when a human last said "still correct".
	DescriptionSource      string     `json:"descriptionSource" db:"description_source"`
	DescriptionAuthoredBy  *string    `json:"descriptionAuthoredBy,omitempty" db:"description_authored_by"`
	DescriptionAuthoredAt  *time.Time `json:"descriptionAuthoredAt,omitempty" db:"description_authored_at"`
	DescriptionAgentModel  *string    `json:"descriptionAgentModel,omitempty" db:"description_agent_model"`
	LastVerifiedAt         *time.Time `json:"lastVerifiedAt,omitempty" db:"last_verified_at"`
	LastVerifiedBy         *string    `json:"lastVerifiedBy,omitempty" db:"last_verified_by"`
	LastVerifiedCommit     *string    `json:"lastVerifiedCommit,omitempty" db:"last_verified_commit"`
	FieldProvenance        any        `json:"fieldProvenance,omitempty" db:"field_provenance"`

	// Creator metadata (migration 011). CreatedBy is the user UUID who first
	// created the node; CreatorEmail is the denormalized email from a LEFT JOIN
	// on users. Empty when the user account has been deleted or for legacy nodes.
	CreatedBy    *string `json:"createdBy,omitempty" db:"created_by"`
	CreatorEmail string  `json:"creatorEmail,omitempty" db:"creator_email"`

	// Domain lifecycle (migration 012). Orthogonal to Status: Status tracks
	// operational progress (PASS/FAIL/IN_PROGRESS/BLOCKED), LifecycleState
	// tracks per-type semantic phase (e.g., REQUIREMENT ACCEPTED,
	// DECISION APPROVED). LifecycleStateChangedAt records when it last flipped.
	LifecycleState          *string    `json:"lifecycleState,omitempty" db:"lifecycle_state"`
	LifecycleStateChangedAt *time.Time `json:"lifecycleStateChangedAt,omitempty" db:"lifecycle_state_changed_at"`
}

type NodeDetail struct {
	Node
	ConnectedEdges   []Edge          `json:"connectedEdges"`
	ConnectedNodeIDs []string        `json:"connectedNodeIds"`
	History          []NodeHistoryEntry `json:"history"`
}

type Edge struct {
	ID         string    `json:"id" db:"id"`
	ProjectID  string    `json:"projectId" db:"project_id"`
	SourceID   string    `json:"sourceId" db:"source_id"`
	TargetID   string    `json:"targetId" db:"target_id"`
	EdgeType   EdgeType  `json:"edgeType" db:"edge_type"`
	Label      *string   `json:"label" db:"label"`
	SourcePort string    `json:"sourcePort" db:"source_port"`
	TargetPort string    `json:"targetPort" db:"target_port"`
	Waypoints  any       `json:"waypoints" db:"waypoints"`
	Metadata   any       `json:"metadata" db:"metadata"`
	CreatedAt  time.Time `json:"createdAt" db:"created_at"`
}

type NodeHistory struct {
	ID        string        `json:"id" db:"id"`
	NodeID    string        `json:"nodeId" db:"node_id"`
	ProjectID string        `json:"projectId" db:"project_id"`
	UserID    string        `json:"userId" db:"user_id"`
	Action    HistoryAction `json:"action" db:"action"`
	FieldName *string       `json:"fieldName" db:"field_name"`
	OldValue  *string       `json:"oldValue" db:"old_value"`
	NewValue  *string       `json:"newValue" db:"new_value"`
	CreatedAt time.Time     `json:"createdAt" db:"created_at"`
}

// NodeHistoryEntry is the frontend/CLI-facing shape of a single audit_log
// row. Action is a plain string because audit_log.action can carry values
// (`imported`, `verified`, `resolved`, ...) that fall outside the smaller
// HistoryAction enum — the client just renders it as text.
type NodeHistoryEntry struct {
	ID        string    `json:"id" db:"id"`
	Action    string    `json:"action" db:"action"`
	FieldName *string   `json:"fieldName" db:"field_name"`
	OldValue  *string   `json:"oldValue" db:"old_value"`
	NewValue  *string   `json:"newValue" db:"new_value"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UserName  string    `json:"userName" db:"display_name"`
}

type APIKey struct {
	ID          string             `json:"id" db:"id"`
	UserID      string             `json:"userId" db:"user_id"`
	Name        string             `json:"name" db:"name"`
	KeyPrefix   string             `json:"keyPrefix" db:"key_prefix"`
	KeyHash     string             `json:"-" db:"key_hash"`
	Kind        APIKeyKind         `json:"kind" db:"kind"`
	Permissions APIKeyPermissions  `json:"permissions" db:"permissions"`
	// ProjectID scopes the key to a single project. NULL = all of the user's
	// projects (v0.5.x behavior). Set on create in v0.6.0+.
	ProjectID   *string            `json:"projectId,omitempty" db:"project_id"`
	LastUsedAt  *time.Time         `json:"lastUsedAt,omitempty" db:"last_used_at"`
	ExpiresAt   *time.Time         `json:"expiresAt,omitempty" db:"expires_at"`
	CreatedAt   time.Time          `json:"createdAt" db:"created_at"`
}

// APIKeyKind classifies the intended caller for an API key.
type APIKeyKind string

const (
	APIKeyKindUserInteractive APIKeyKind = "user_interactive"
	APIKeyKindAgent           APIKeyKind = "agent"
	APIKeyKindService         APIKeyKind = "service"
)

// APIKeyPermissions controls which mutation classes a key is allowed to perform.
// Stored as JSONB in api_keys.permissions. See migration 008.
type APIKeyPermissions struct {
	Read            bool `json:"read"`
	WriteStructural bool `json:"write_structural"`
	WriteSemantic   bool `json:"write_semantic"`
	WriteMeta       bool `json:"write_meta"`
	Verify          bool `json:"verify"`
	Delete          bool `json:"delete"`
	Suggest         bool `json:"suggest"`
}

// Scan implements sql.Scanner so pgx can hydrate the struct from JSONB.
// An empty / NULL / "{}" JSONB value is interpreted as "no explicit overrides"
// and resolved to the all-true preset — matching the intent of migration 008's
// backfill SQL. Returning an all-false zero value here would silently 403 every
// existing API key the instant Scan saw a not-yet-backfilled row.
func (p *APIKeyPermissions) Scan(src any) error {
	if src == nil {
		*p = DefaultPermissions(APIKeyKindUserInteractive)
		return nil
	}
	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("APIKeyPermissions: cannot scan %T", src)
	}
	if len(data) == 0 || string(data) == "{}" {
		*p = DefaultPermissions(APIKeyKindUserInteractive)
		return nil
	}
	return json.Unmarshal(data, p)
}

// Value implements driver.Valuer so the struct can be inserted as JSONB.
func (p APIKeyPermissions) Value() (driver.Value, error) {
	return json.Marshal(p)
}

// DefaultPermissions returns the safe-by-default preset for a given key kind.
// agent kind blocks semantic writes and verify; the UI may override per-checkbox.
func DefaultPermissions(kind APIKeyKind) APIKeyPermissions {
	switch kind {
	case APIKeyKindAgent:
		return APIKeyPermissions{
			Read: true, WriteStructural: true, WriteSemantic: false,
			WriteMeta: true, Verify: false, Delete: true, Suggest: true,
		}
	case APIKeyKindService:
		return APIKeyPermissions{
			Read: true, WriteStructural: true, WriteSemantic: true,
			WriteMeta: true, Verify: false, Delete: true, Suggest: true,
		}
	default: // user_interactive
		return APIKeyPermissions{
			Read: true, WriteStructural: true, WriteSemantic: true,
			WriteMeta: true, Verify: true, Delete: true, Suggest: true,
		}
	}
}

// Allows checks whether the given mutation kind / action combo is permitted.
// mutationKind: "structural" | "semantic" | "meta".
// action: "read" | "write" | "delete" | "verify" | "suggest".
func (p APIKeyPermissions) Allows(mutationKind, action string) bool {
	switch action {
	case "read":
		return p.Read
	case "delete":
		return p.Delete
	case "verify":
		return p.Verify
	case "suggest":
		return p.Suggest
	case "write":
		switch mutationKind {
		case "structural":
			return p.WriteStructural
		case "semantic":
			return p.WriteSemantic
		case "meta":
			return p.WriteMeta
		}
	}
	return false
}

type ImpactResult struct {
	ChangedNodes  []Node `json:"changedNodes"`
	ImpactedNodes []Node `json:"impactedNodes"`
	FailNodes     []Node `json:"failNodes"`
	ImpactEdges   []Edge `json:"impactEdges"`
}

// AuditLog captures one write event with the 6 orthogonal dimensions defined
// in migration 006. Insert path lives in service/audit.go; read path in
// repository/audit.go.
type AuditLog struct {
	ID              string     `json:"id" db:"id"`
	ProjectID       string     `json:"projectId" db:"project_id"`
	EntityType      string     `json:"entityType" db:"entity_type"`
	EntityID        *string    `json:"entityId,omitempty" db:"entity_id"`
	Action          string     `json:"action" db:"action"`
	FieldName       *string    `json:"fieldName,omitempty" db:"field_name"`
	OldValue        *string    `json:"oldValue,omitempty" db:"old_value"`
	NewValue        *string    `json:"newValue,omitempty" db:"new_value"`
	MutationKind    string     `json:"mutationKind" db:"mutation_kind"`
	UserID          *string    `json:"userId,omitempty" db:"user_id"`
	APIKeyID        *string    `json:"apiKeyId,omitempty" db:"api_key_id"`
	ActorKind       string     `json:"actorKind" db:"actor_kind"`
	ClientType      *string    `json:"clientType,omitempty" db:"client_type"`
	ClientVersion   *string    `json:"clientVersion,omitempty" db:"client_version"`
	AgentModel      *string    `json:"agentModel,omitempty" db:"agent_model"`
	AgentSessionID  *string    `json:"agentSessionId,omitempty" db:"agent_session_id"`
	Trigger         *string    `json:"trigger,omitempty" db:"trigger"`
	BatchID         *string    `json:"batchId,omitempty" db:"batch_id"`
	CodeCommit      *string    `json:"codeCommit,omitempty" db:"code_commit"`
	SourcePath      *string    `json:"sourcePath,omitempty" db:"source_path"`
	Confidence      *string    `json:"confidence,omitempty" db:"confidence"`
	Metadata        any        `json:"metadata,omitempty" db:"metadata"`
	CreatedAt       time.Time  `json:"createdAt" db:"created_at"`
}

// NodeSuggestion is an agent-proposed update queued for human review.
// Created via thask.node.suggest_update; reviewed via thask.suggestions.decide.
type NodeSuggestion struct {
	ID              string     `json:"id" db:"id"`
	ProjectID       string     `json:"projectId" db:"project_id"`
	NodeID          string     `json:"nodeId" db:"node_id"`
	FieldName       string     `json:"fieldName" db:"field_name"`
	ProposedValue   string     `json:"proposedValue" db:"proposed_value"`
	CurrentValue    *string    `json:"currentValue,omitempty" db:"current_value"`
	Rationale       *string    `json:"rationale,omitempty" db:"rationale"`
	Evidence        any        `json:"evidence,omitempty" db:"evidence"`
	ProposedBy      string     `json:"proposedBy" db:"proposed_by"`
	AgentModel      *string    `json:"agentModel,omitempty" db:"agent_model"`
	AgentSessionID  *string    `json:"agentSessionId,omitempty" db:"agent_session_id"`
	Status          string     `json:"status" db:"status"`
	DecidedBy       *string    `json:"decidedBy,omitempty" db:"decided_by"`
	DecidedAt       *time.Time `json:"decidedAt,omitempty" db:"decided_at"`
	DecidedReason   *string    `json:"decidedReason,omitempty" db:"decided_reason"`
	CreatedAt       time.Time  `json:"createdAt" db:"created_at"`
}

// NodeComment is a threaded discussion note attached to a node (v0.6.0 B-1).
// parent_id is self-referential to model one-level threads. resolved_at is
// set when a reviewer marks the thread resolved.
type NodeComment struct {
	ID         string     `json:"id" db:"id"`
	NodeID     string     `json:"nodeId" db:"node_id"`
	ProjectID  string     `json:"projectId" db:"project_id"`
	AuthorID   string     `json:"authorId" db:"author_id"`
	ParentID   *string    `json:"parentId,omitempty" db:"parent_id"`
	Body       string     `json:"body" db:"body"`
	ResolvedAt *time.Time `json:"resolvedAt,omitempty" db:"resolved_at"`
	ResolvedBy *string    `json:"resolvedBy,omitempty" db:"resolved_by"`
	CreatedAt  time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt  time.Time  `json:"updatedAt" db:"updated_at"`
}

// NodeAttachment is a file attached to a node (v0.6.0 B-2). storage_key is a
// relative path inside THASK_ATTACHMENT_DIR (local filesystem) — swap for an
// S3 object key when MinIO backend lands.
type NodeAttachment struct {
	ID         string    `json:"id" db:"id"`
	NodeID     string    `json:"nodeId" db:"node_id"`
	ProjectID  string    `json:"projectId" db:"project_id"`
	Filename   string    `json:"filename" db:"filename"`
	MimeType   string    `json:"mimeType" db:"mime_type"`
	SizeBytes  int64     `json:"sizeBytes" db:"size_bytes"`
	StorageKey string    `json:"-" db:"storage_key"`
	SHA256     string    `json:"sha256" db:"sha256"`
	UploadedBy string    `json:"uploadedBy" db:"uploaded_by"`
	CreatedAt  time.Time `json:"createdAt" db:"created_at"`
}

// ProjectTag is canonical metadata for a tag used across a project (v0.6.0 B-3).
// nodes.tags TEXT[] stays the primary storage — this table augments it so the
// UI can offer autocomplete + consistent styling.
type ProjectTag struct {
	ProjectID   string    `json:"projectId" db:"project_id"`
	Tag         string    `json:"tag" db:"tag"`
	Color       *string   `json:"color,omitempty" db:"color"`
	Description *string   `json:"description,omitempty" db:"description"`
	CreatedAt   time.Time `json:"createdAt" db:"created_at"`
	CreatedBy   *string   `json:"createdBy,omitempty" db:"created_by"`
}
