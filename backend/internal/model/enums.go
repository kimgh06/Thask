package model

type TeamRole string

const (
	TeamRoleOwner  TeamRole = "owner"
	TeamRoleAdmin  TeamRole = "admin"
	TeamRoleMember TeamRole = "member"
	TeamRoleViewer TeamRole = "viewer"
)

// Level returns numeric level for hierarchy comparison.
// owner(4) > admin(3) > member(2) > viewer(1)
func (r TeamRole) Level() int {
	switch r {
	case TeamRoleOwner:
		return 4
	case TeamRoleAdmin:
		return 3
	case TeamRoleMember:
		return 2
	case TeamRoleViewer:
		return 1
	default:
		return 0
	}
}

func (r TeamRole) AtLeast(min TeamRole) bool {
	return r.Level() >= min.Level()
}

type NodeType string

const (
	NodeTypeFlow        NodeType = "FLOW"
	NodeTypeBranch      NodeType = "BRANCH"
	NodeTypeTask        NodeType = "TASK"
	NodeTypeBug         NodeType = "BUG"
	NodeTypeAPI         NodeType = "API"
	NodeTypeUI          NodeType = "UI"
	NodeTypeGroup       NodeType = "GROUP"
	NodeTypeRequirement NodeType = "REQUIREMENT"
	NodeTypeDecision    NodeType = "DECISION"
	NodeTypeExperiment  NodeType = "EXPERIMENT"
	NodeTypePerson      NodeType = "PERSON"
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

	// v0.6.0 Knowledge OS edges. `realizes`/`conflicts` link REQUIREMENTs,
	// `drives`/`supersedes` link DECISIONs, `tests`/`produced` link EXPERIMENTs,
	// and `owns`/`decided`/`reported` link PERSONs to work/decisions/bugs.
	EdgeTypeRealizes   EdgeType = "realizes"
	EdgeTypeConflicts  EdgeType = "conflicts"
	EdgeTypeDrives     EdgeType = "drives"
	EdgeTypeSupersedes EdgeType = "supersedes"
	EdgeTypeTests      EdgeType = "tests"
	EdgeTypeProduced   EdgeType = "produced"
	EdgeTypeOwns       EdgeType = "owns"
	EdgeTypeDecided    EdgeType = "decided"
	EdgeTypeReported   EdgeType = "reported"
)

type ProjectRole string

const (
	ProjectRoleEditor ProjectRole = "editor"
	ProjectRoleViewer ProjectRole = "viewer"
)

// ToTeamRole maps a ProjectRole to its equivalent TeamRole for authorization.
func (r ProjectRole) ToTeamRole() TeamRole {
	switch r {
	case ProjectRoleEditor:
		return TeamRoleMember
	case ProjectRoleViewer:
		return TeamRoleViewer
	default:
		return ""
	}
}

type HistoryAction string

const (
	HistoryActionCreated       HistoryAction = "created"
	HistoryActionUpdated       HistoryAction = "updated"
	HistoryActionDeleted       HistoryAction = "deleted"
	HistoryActionStatusChanged HistoryAction = "status_changed"
)
