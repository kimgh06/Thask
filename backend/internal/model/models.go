package model

import "time"

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
	CreatedAt   time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time `json:"updatedAt" db:"updated_at"`
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
}

type NodeDetail struct {
	Node
	ConnectedEdges   []Edge          `json:"connectedEdges"`
	ConnectedNodeIDs []string        `json:"connectedNodeIds"`
	History          []NodeHistoryEntry `json:"history"`
}

type Edge struct {
	ID        string   `json:"id" db:"id"`
	ProjectID string   `json:"projectId" db:"project_id"`
	SourceID  string   `json:"sourceId" db:"source_id"`
	TargetID  string   `json:"targetId" db:"target_id"`
	EdgeType  EdgeType `json:"edgeType" db:"edge_type"`
	Label     *string  `json:"label" db:"label"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
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

type NodeHistoryEntry struct {
	ID        string        `json:"id" db:"id"`
	Action    HistoryAction `json:"action" db:"action"`
	FieldName *string       `json:"fieldName" db:"field_name"`
	OldValue  *string       `json:"oldValue" db:"old_value"`
	NewValue  *string       `json:"newValue" db:"new_value"`
	CreatedAt time.Time     `json:"createdAt" db:"created_at"`
	UserName  string        `json:"userName" db:"display_name"`
}

type ImpactResult struct {
	ChangedNodes  []Node `json:"changedNodes"`
	ImpactedNodes []Node `json:"impactedNodes"`
	FailNodes     []Node `json:"failNodes"`
	ImpactEdges   []Edge `json:"impactEdges"`
}
