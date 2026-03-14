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
	AssigneeID  *string  `json:"assigneeId" validate:"omitempty,uuid"`
	Tags        []string `json:"tags"`
	ParentID    *string  `json:"parentId" validate:"omitempty,uuid"`
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

type CreateEdgeRequest struct {
	SourceID string  `json:"sourceId" validate:"required,uuid"`
	TargetID string  `json:"targetId" validate:"required,uuid"`
	EdgeType string  `json:"edgeType" validate:"omitempty,oneof=depends_on blocks related parent_child triggers"`
	Label    *string `json:"label" validate:"omitempty,max=100"`
}

type UpdateEdgeRequest struct {
	EdgeType *string `json:"edgeType" validate:"omitempty,oneof=depends_on blocks related parent_child triggers"`
	Label    *string `json:"label" validate:"omitempty,max=100"`
}
