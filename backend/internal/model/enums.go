package model

type TeamRole string

const (
	TeamRoleOwner  TeamRole = "owner"
	TeamRoleAdmin  TeamRole = "admin"
	TeamRoleMember TeamRole = "member"
	TeamRoleViewer TeamRole = "viewer"
)

type NodeType string

const (
	NodeTypeFlow   NodeType = "FLOW"
	NodeTypeBranch NodeType = "BRANCH"
	NodeTypeTask   NodeType = "TASK"
	NodeTypeBug    NodeType = "BUG"
	NodeTypeAPI    NodeType = "API"
	NodeTypeUI     NodeType = "UI"
	NodeTypeGroup  NodeType = "GROUP"
)

type NodeStatus string

const (
	NodeStatusPass       NodeStatus = "PASS"
	NodeStatusFail       NodeStatus = "FAIL"
	NodeStatusInProgress NodeStatus = "IN_PROGRESS"
	NodeStatusBlocked    NodeStatus = "BLOCKED"
)

type EdgeType string

const (
	EdgeTypeDependsOn   EdgeType = "depends_on"
	EdgeTypeBlocks      EdgeType = "blocks"
	EdgeTypeRelated     EdgeType = "related"
	EdgeTypeParentChild EdgeType = "parent_child"
	EdgeTypeTriggers    EdgeType = "triggers"
)

type HistoryAction string

const (
	HistoryActionCreated       HistoryAction = "created"
	HistoryActionUpdated       HistoryAction = "updated"
	HistoryActionDeleted       HistoryAction = "deleted"
	HistoryActionStatusChanged HistoryAction = "status_changed"
)
