package dto

type RegisterRequest struct {
	Email       string `json:"email" validate:"required,email"`
	Password    string `json:"password" validate:"required,min=8"`
	DisplayName string `json:"displayName" validate:"required,min=1,max=100"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=1"`
}

type CreateTeamRequest struct {
	Name string `json:"name" validate:"required,min=1,max=100"`
	Slug string `json:"slug" validate:"required,min=1,max=100,slug"`
}

type UpdateTeamRequest struct {
	Name string `json:"name" validate:"required,min=1,max=100"`
}

type InviteMemberRequest struct {
	Email string `json:"email" validate:"required,email"`
	Role  string `json:"role" validate:"omitempty,oneof=admin member viewer"`
}

type CreateProjectRequest struct {
	Name        string  `json:"name" validate:"required,min=1,max=200"`
	Description *string `json:"description" validate:"omitempty,max=2000"`
}

type UpdateProjectRequest struct {
	Name        *string `json:"name" validate:"omitempty,min=1,max=200"`
	Description *string `json:"description" validate:"omitempty,max=2000"`
}

type CreateNodeRequest struct {
	Type        string   `json:"type" validate:"required,oneof=FLOW BRANCH TASK BUG API UI GROUP"`
	Title       string   `json:"title" validate:"required,min=1,max=300"`
	Description *string  `json:"description" validate:"omitempty,max=5000"`
	Status      string   `json:"status" validate:"omitempty,oneof=PASS FAIL IN_PROGRESS BLOCKED"`
	AssigneeID  *string  `json:"assigneeId" validate:"omitempty,uuid"`
	Tags        []string `json:"tags"`
	PositionX   float64  `json:"positionX"`
	PositionY   float64  `json:"positionY"`
	Width       *float64 `json:"width" validate:"omitempty,min=80"`
	Height      *float64 `json:"height" validate:"omitempty,min=50"`
}

type UpdateNodeRequest struct {
	Type        *string  `json:"type" validate:"omitempty,oneof=FLOW BRANCH TASK BUG API UI GROUP"`
	Title       *string  `json:"title" validate:"omitempty,min=1,max=300"`
	Description *string  `json:"description" validate:"omitempty,max=5000"`
	Status      *string  `json:"status" validate:"omitempty,oneof=PASS FAIL IN_PROGRESS BLOCKED"`
	AssigneeID  *string  `json:"assigneeId"`
	Tags        []string `json:"tags"`
	ParentID    *string  `json:"parentId"`
	Width       *float64 `json:"width" validate:"omitempty,min=80"`
	Height      *float64 `json:"height" validate:"omitempty,min=50"`
}

type BatchPositionItem struct {
	ID     string   `json:"id" validate:"required,uuid"`
	X      float64  `json:"x"`
	Y      float64  `json:"y"`
	Width  *float64 `json:"width" validate:"omitempty,min=80"`
	Height *float64 `json:"height" validate:"omitempty,min=50"`
}

type BatchPositionRequest struct {
	Positions []BatchPositionItem `json:"positions" validate:"required,dive"`
}

type BatchDeleteRequest struct {
	IDs []string `json:"ids" validate:"required,min=1,dive,uuid"`
}

type BatchStatusRequest struct {
	IDs    []string `json:"ids" validate:"required,min=1,dive,uuid"`
	Status string   `json:"status" validate:"required,oneof=PASS FAIL IN_PROGRESS BLOCKED"`
}

type CreateEdgeRequest struct {
	SourceID   string  `json:"sourceId" validate:"required,uuid"`
	TargetID   string  `json:"targetId" validate:"required,uuid"`
	EdgeType   string  `json:"edgeType" validate:"omitempty,oneof=depends_on blocks related parent_child triggers"`
	Label      *string `json:"label" validate:"omitempty,max=100"`
	SourcePort string  `json:"sourcePort" validate:"omitempty,oneof=auto N NE E SE S SW W NW"`
	TargetPort string  `json:"targetPort" validate:"omitempty,oneof=auto N NE E SE S SW W NW"`
	Waypoints  any     `json:"waypoints"`
}

type UpdateEdgeRequest struct {
	EdgeType   *string `json:"edgeType" validate:"omitempty,oneof=depends_on blocks related parent_child triggers"`
	Label      *string `json:"label" validate:"omitempty,max=100"`
	SourcePort *string `json:"sourcePort" validate:"omitempty,oneof=auto N NE E SE S SW W NW"`
	TargetPort *string `json:"targetPort" validate:"omitempty,oneof=auto N NE E SE S SW W NW"`
	Waypoints  any     `json:"waypoints"`
}

type UpdateMemberRoleRequest struct {
	Role string `json:"role" validate:"required,oneof=admin member viewer"`
}

type TransferOwnershipRequest struct {
	UserID string `json:"userId" validate:"required,uuid"`
}

type UpdateSharingRequest struct {
	LinkSharing string `json:"linkSharing" validate:"required,oneof=off viewer editor"`
}

type AddProjectMemberRequest struct {
	Email string `json:"email" validate:"required,email"`
	Role  string `json:"role" validate:"required,oneof=editor viewer"`
}

type UpdateProjectMemberRequest struct {
	Role string `json:"role" validate:"required,oneof=editor viewer"`
}

type LayoutRequest struct {
	Algorithm     string `json:"algorithm" validate:"omitempty,oneof=dagre grid"`
	PreserveEdges bool   `json:"preserveEdges"`
}

type CreateAPIKeyRequest struct {
	Name        string                       `json:"name" validate:"required,min=1,max=100"`
	Kind        string                       `json:"kind" validate:"omitempty,oneof=user_interactive agent service"`
	Permissions *map[string]bool             `json:"permissions"`
	ExpiresIn   *int                         `json:"expiresIn" validate:"omitempty,min=1,max=365"`
}

type ImportGraphRequest struct {
	Mode  string           `json:"mode" validate:"required,oneof=replace merge"`
	Nodes []ImportNodeItem `json:"nodes" validate:"required,dive"`
	Edges []ImportEdgeItem `json:"edges" validate:"dive"`
}

type ImportNodeItem struct {
	ID          string   `json:"id"`
	Type        string   `json:"type" validate:"required,oneof=FLOW BRANCH TASK BUG API UI GROUP"`
	Title       string   `json:"title" validate:"required,min=1,max=300"`
	Description *string  `json:"description"`
	Status      string   `json:"status" validate:"omitempty,oneof=PASS FAIL IN_PROGRESS BLOCKED"`
	Tags        []string `json:"tags"`
	ParentID    *string  `json:"parentId"`
	PositionX   float64  `json:"positionX"`
	PositionY   float64  `json:"positionY"`
	Width       *float64 `json:"width"`
	Height      *float64 `json:"height"`
}

type ImportEdgeItem struct {
	SourceID string  `json:"sourceId" validate:"required"`
	TargetID string  `json:"targetId" validate:"required"`
	EdgeType string  `json:"edgeType" validate:"omitempty,oneof=depends_on blocks related parent_child triggers"`
	Label    *string `json:"label"`
}

// SuggestNodeUpdateRequest is the body for POST /api/projects/:pid/nodes/:nid/suggestions.
// Agents call this instead of mutating description directly.
type SuggestNodeUpdateRequest struct {
	FieldName     string `json:"fieldName" validate:"omitempty,oneof=description title tags"`
	ProposedValue string `json:"proposedValue" validate:"required"`
	Rationale     string `json:"rationale" validate:"omitempty,max=2000"`
	Evidence      any    `json:"evidence"` // { codeCommit, sourcePaths[], confidence }
}

// DecideSuggestionRequest is the body for PATCH /api/projects/:pid/suggestions/:sid.
type DecideSuggestionRequest struct {
	Status string `json:"status" validate:"required,oneof=accepted rejected"`
	Reason string `json:"reason" validate:"omitempty,max=2000"`
}

// VerifyNodeRequest is the body for POST /api/projects/:pid/nodes/:nid/verify.
// commit is optional but recommended — the CLI/MCP fills it from git.
type VerifyNodeRequest struct {
	Commit string `json:"commit" validate:"omitempty,max=64"`
}

// BatchNodeUpdateRequest is the body for PATCH /api/projects/:pid/nodes/batch-update.
// Each item carries the target node id plus any subset of fields to change.
// Use this instead of looping node.update — it cuts agent context overhead and
// runs the whole batch under one transaction so cycle / permission failures
// roll back atomically.
type BatchNodeUpdateRequest struct {
	Updates []BatchNodeUpdateItem `json:"updates" validate:"required,min=1,max=200,dive"`
}

type BatchNodeUpdateItem struct {
	NodeID      string   `json:"nodeId"      validate:"required,uuid"`
	Type        *string  `json:"type,omitempty"        validate:"omitempty,oneof=FLOW BRANCH TASK BUG API UI GROUP"`
	Title       *string  `json:"title,omitempty"`
	Description *string  `json:"description,omitempty"`
	Status      *string  `json:"status,omitempty"      validate:"omitempty,oneof=PASS FAIL IN_PROGRESS BLOCKED"`
	Tags        []string `json:"tags,omitempty"`
	AssigneeID  *string  `json:"assigneeId,omitempty"`
	ParentID    *string  `json:"parentId,omitempty"`
}

// BatchEdgeCreateRequest is the body for POST /api/projects/:pid/edges/batch-create.
// Use this instead of looping edge.create. Self-references, duplicates, and
// edges referencing missing nodes are skipped with reasons; permission errors
// reject the whole batch.
type BatchEdgeCreateRequest struct {
	Edges []BatchEdgeCreateItem `json:"edges" validate:"required,min=1,max=500,dive"`
}

type BatchEdgeCreateItem struct {
	SourceID   string   `json:"sourceId" validate:"required,uuid"`
	TargetID   string   `json:"targetId" validate:"required,uuid"`
	EdgeType   string   `json:"edgeType" validate:"omitempty,oneof=depends_on blocks related parent_child triggers"`
	Label      *string  `json:"label"`
	SourcePort string   `json:"sourcePort"`
	TargetPort string   `json:"targetPort"`
	Waypoints  any      `json:"waypoints"`
}

// BatchEdgeDeleteRequest is the body for POST /api/projects/:pid/edges/batch-delete.
// Missing edge ids are skipped (returned in skipped[]), permission errors reject the batch.
type BatchEdgeDeleteRequest struct {
	EdgeIDs []string `json:"edgeIds" validate:"required,min=1,max=500,dive,uuid"`
}
