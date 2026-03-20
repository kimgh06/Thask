package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/thask/backend/internal/dto"
	mw "github.com/thask/backend/internal/middleware"
	"github.com/thask/backend/internal/model"
	"github.com/thask/backend/internal/repository"
)

type TeamHandler struct {
	teamRepo    *repository.TeamRepo
	projectRepo *repository.ProjectRepo
	userRepo    *repository.UserRepo
}

func NewTeamHandler(teamRepo *repository.TeamRepo, projectRepo *repository.ProjectRepo, userRepo *repository.UserRepo) *TeamHandler {
	return &TeamHandler{teamRepo: teamRepo, projectRepo: projectRepo, userRepo: userRepo}
}

func (h *TeamHandler) List(c echo.Context) error {
	ctx := c.Request().Context()
	userID := mw.GetUserID(c)

	teams, err := h.teamRepo.FindByUserID(ctx, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to fetch teams"))
	}
	if teams == nil {
		return c.JSON(http.StatusOK, dto.OK([]any{}))
	}

	teamIDs := make([]string, len(teams))
	for i, t := range teams {
		teamIDs[i] = t.ID
	}

	projects, err := h.projectRepo.FindByTeamIDs(ctx, teamIDs)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to fetch projects"))
	}

	result := make([]model.TeamWithProjects, len(teams))
	for i, t := range teams {
		tp := model.TeamWithProjects{Team: t, Projects: []model.Project{}}
		for _, p := range projects {
			if p.TeamID == t.ID {
				tp.Projects = append(tp.Projects, p)
			}
		}
		result[i] = tp
	}

	return c.JSON(http.StatusOK, dto.OK(result))
}

func (h *TeamHandler) Create(c echo.Context) error {
	var req dto.CreateTeamRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err("Invalid request body"))
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err(err.Error()))
	}

	ctx := c.Request().Context()
	userID := mw.GetUserID(c)

	if existing, _ := h.teamRepo.FindBySlug(ctx, req.Slug); existing != nil {
		return c.JSON(http.StatusConflict, dto.Err("Team slug already taken"))
	}

	team, err := h.teamRepo.Create(ctx, req.Name, req.Slug, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to create team"))
	}

	_ = h.teamRepo.AddMember(ctx, team.ID, userID, model.TeamRoleOwner)

	return c.JSON(http.StatusCreated, dto.OK(team))
}

// --- Below: TeamAccess middleware provides team_id + team_role in context ---

func (h *TeamHandler) GetBySlug(c echo.Context) error {
	ctx := c.Request().Context()
	slug := c.Param("teamSlug")

	team, err := h.teamRepo.FindBySlug(ctx, slug)
	if err != nil {
		return c.JSON(http.StatusNotFound, dto.Err("Team not found"))
	}

	return c.JSON(http.StatusOK, dto.OK(team))
}

func (h *TeamHandler) Update(c echo.Context) error {
	var req dto.UpdateTeamRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err("Invalid request body"))
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err(err.Error()))
	}

	ctx := c.Request().Context()
	teamID := mw.GetTeamID(c)

	updated, err := h.teamRepo.Update(ctx, teamID, req.Name)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to update team"))
	}

	return c.JSON(http.StatusOK, dto.OK(updated))
}

func (h *TeamHandler) Delete(c echo.Context) error {
	ctx := c.Request().Context()
	teamID := mw.GetTeamID(c)

	if err := h.teamRepo.Delete(ctx, teamID); err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to delete team"))
	}

	return c.JSON(http.StatusOK, dto.OK(dto.SuccessResponse{Success: true}))
}

func (h *TeamHandler) ListMembers(c echo.Context) error {
	ctx := c.Request().Context()
	teamID := mw.GetTeamID(c)

	members, err := h.teamRepo.GetMembers(ctx, teamID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to fetch members"))
	}

	return c.JSON(http.StatusOK, dto.OK(members))
}

func (h *TeamHandler) InviteMember(c echo.Context) error {
	var req dto.InviteMemberRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err("Invalid request body"))
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err(err.Error()))
	}

	ctx := c.Request().Context()
	teamID := mw.GetTeamID(c)
	callerRole := mw.GetTeamRole(c)

	invitee, err := h.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return c.JSON(http.StatusNotFound, dto.Err("User not found"))
	}

	role := model.TeamRole(req.Role)
	if role == "" {
		role = model.TeamRoleMember
	}

	// Admin cannot invite as admin or owner
	if callerRole == model.TeamRoleAdmin && role.AtLeast(model.TeamRoleAdmin) {
		return c.JSON(http.StatusForbidden, dto.Err("Admins cannot invite as admin or owner"))
	}

	if err := h.teamRepo.AddMember(ctx, teamID, invitee.ID, role); err != nil {
		return c.JSON(http.StatusConflict, dto.Err("User already a member"))
	}

	return c.JSON(http.StatusCreated, dto.OK(dto.SuccessResponse{Success: true}))
}

func (h *TeamHandler) ListProjects(c echo.Context) error {
	ctx := c.Request().Context()
	teamID := mw.GetTeamID(c)

	projects, err := h.projectRepo.FindByTeamID(ctx, teamID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to fetch projects"))
	}
	if projects == nil {
		projects = []model.Project{}
	}

	return c.JSON(http.StatusOK, dto.OK(projects))
}

func (h *TeamHandler) CreateProject(c echo.Context) error {
	var req dto.CreateProjectRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err("Invalid request body"))
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err(err.Error()))
	}

	ctx := c.Request().Context()
	teamID := mw.GetTeamID(c)
	userID := mw.GetUserID(c)

	project, err := h.projectRepo.Create(ctx, teamID, req.Name, req.Description, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to create project"))
	}

	return c.JSON(http.StatusCreated, dto.OK(project))
}

// --- P1: Team Member Management ---

func (h *TeamHandler) UpdateMemberRole(c echo.Context) error {
	var req dto.UpdateMemberRoleRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err("Invalid request body"))
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err(err.Error()))
	}

	ctx := c.Request().Context()
	teamID := mw.GetTeamID(c)
	callerID := mw.GetUserID(c)
	callerRole := mw.GetTeamRole(c)
	targetUserID := c.Param("userId")
	newRole := model.TeamRole(req.Role)

	if callerID == targetUserID {
		return c.JSON(http.StatusBadRequest, dto.Err("Use /transfer to change ownership or /leave to leave"))
	}

	targetRole, err := h.teamRepo.GetMemberRole(ctx, teamID, targetUserID)
	if err != nil {
		return c.JSON(http.StatusNotFound, dto.Err("Member not found"))
	}

	if targetRole == model.TeamRoleOwner {
		return c.JSON(http.StatusForbidden, dto.Err("Cannot change owner's role"))
	}

	if newRole == model.TeamRoleOwner {
		return c.JSON(http.StatusBadRequest, dto.Err("Use /transfer to transfer ownership"))
	}

	// Admin can only manage member/viewer
	if callerRole == model.TeamRoleAdmin {
		if targetRole == model.TeamRoleAdmin {
			return c.JSON(http.StatusForbidden, dto.Err("Admins cannot manage other admins"))
		}
		if newRole == model.TeamRoleAdmin {
			return c.JSON(http.StatusForbidden, dto.Err("Admins cannot promote to admin"))
		}
	}

	if err := h.teamRepo.UpdateMemberRole(ctx, teamID, targetUserID, newRole); err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to update role"))
	}

	return c.JSON(http.StatusOK, dto.OK(dto.SuccessResponse{Success: true}))
}

func (h *TeamHandler) RemoveMember(c echo.Context) error {
	ctx := c.Request().Context()
	teamID := mw.GetTeamID(c)
	callerID := mw.GetUserID(c)
	callerRole := mw.GetTeamRole(c)
	targetUserID := c.Param("userId")

	if callerID == targetUserID {
		return c.JSON(http.StatusBadRequest, dto.Err("Use /leave to remove yourself"))
	}

	targetRole, err := h.teamRepo.GetMemberRole(ctx, teamID, targetUserID)
	if err != nil {
		return c.JSON(http.StatusNotFound, dto.Err("Member not found"))
	}

	if targetRole == model.TeamRoleOwner {
		return c.JSON(http.StatusForbidden, dto.Err("Cannot remove owner"))
	}

	if callerRole == model.TeamRoleAdmin && targetRole == model.TeamRoleAdmin {
		return c.JSON(http.StatusForbidden, dto.Err("Admins cannot remove other admins"))
	}

	if err := h.teamRepo.RemoveMember(ctx, teamID, targetUserID); err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to remove member"))
	}

	return c.JSON(http.StatusOK, dto.OK(dto.SuccessResponse{Success: true}))
}

func (h *TeamHandler) Leave(c echo.Context) error {
	ctx := c.Request().Context()
	teamID := mw.GetTeamID(c)
	userID := mw.GetUserID(c)
	role := mw.GetTeamRole(c)

	if role == model.TeamRoleOwner {
		count, err := h.teamRepo.CountByRole(ctx, teamID, model.TeamRoleOwner)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, dto.Err("Failed to check owner count"))
		}
		if count <= 1 {
			return c.JSON(http.StatusBadRequest, dto.Err("Cannot leave: you are the sole owner. Transfer ownership first."))
		}
	}

	if err := h.teamRepo.RemoveMember(ctx, teamID, userID); err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to leave team"))
	}

	return c.JSON(http.StatusOK, dto.OK(dto.SuccessResponse{Success: true}))
}

func (h *TeamHandler) TransferOwnership(c echo.Context) error {
	var req dto.TransferOwnershipRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err("Invalid request body"))
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err(err.Error()))
	}

	ctx := c.Request().Context()
	teamID := mw.GetTeamID(c)
	callerID := mw.GetUserID(c)

	if callerID == req.UserID {
		return c.JSON(http.StatusBadRequest, dto.Err("Cannot transfer ownership to yourself"))
	}

	if _, err := h.teamRepo.GetMemberRole(ctx, teamID, req.UserID); err != nil {
		return c.JSON(http.StatusNotFound, dto.Err("Target user is not a team member"))
	}

	if err := h.teamRepo.TransferOwnership(ctx, teamID, callerID, req.UserID); err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to transfer ownership"))
	}

	return c.JSON(http.StatusOK, dto.OK(dto.SuccessResponse{Success: true}))
}
