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

	// Check slug uniqueness
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

func (h *TeamHandler) GetBySlug(c echo.Context) error {
	ctx := c.Request().Context()
	slug := c.Param("teamSlug")
	userID := mw.GetUserID(c)

	team, err := h.teamRepo.FindBySlug(ctx, slug)
	if err != nil {
		return c.JSON(http.StatusNotFound, dto.Err("Team not found"))
	}

	isMember, _ := h.teamRepo.IsMember(ctx, team.ID, userID)
	if !isMember {
		return c.JSON(http.StatusNotFound, dto.Err("Team not found"))
	}

	return c.JSON(http.StatusOK, dto.OK(team))
}

func (h *TeamHandler) Delete(c echo.Context) error {
	ctx := c.Request().Context()
	slug := c.Param("teamSlug")
	userID := mw.GetUserID(c)

	team, err := h.teamRepo.FindBySlug(ctx, slug)
	if err != nil {
		return c.JSON(http.StatusNotFound, dto.Err("Team not found"))
	}

	isMember, _ := h.teamRepo.IsMember(ctx, team.ID, userID)
	if !isMember {
		return c.JSON(http.StatusNotFound, dto.Err("Team not found"))
	}

	if err := h.teamRepo.Delete(ctx, team.ID); err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to delete team"))
	}

	return c.JSON(http.StatusOK, dto.OK(dto.SuccessResponse{Success: true}))
}

func (h *TeamHandler) ListMembers(c echo.Context) error {
	ctx := c.Request().Context()
	slug := c.Param("teamSlug")
	userID := mw.GetUserID(c)

	team, err := h.teamRepo.FindBySlug(ctx, slug)
	if err != nil {
		return c.JSON(http.StatusNotFound, dto.Err("Team not found"))
	}

	// Fix: verify requesting user is a member
	isMember, _ := h.teamRepo.IsMember(ctx, team.ID, userID)
	if !isMember {
		return c.JSON(http.StatusNotFound, dto.Err("Team not found"))
	}

	members, err := h.teamRepo.GetMembers(ctx, team.ID)
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
	slug := c.Param("teamSlug")
	userID := mw.GetUserID(c)

	team, err := h.teamRepo.FindBySlug(ctx, slug)
	if err != nil {
		return c.JSON(http.StatusNotFound, dto.Err("Team not found"))
	}

	isMember, _ := h.teamRepo.IsMember(ctx, team.ID, userID)
	if !isMember {
		return c.JSON(http.StatusNotFound, dto.Err("Team not found"))
	}

	invitee, err := h.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return c.JSON(http.StatusNotFound, dto.Err("User not found"))
	}

	role := model.TeamRole(req.Role)
	if role == "" {
		role = model.TeamRoleMember
	}

	if err := h.teamRepo.AddMember(ctx, team.ID, invitee.ID, role); err != nil {
		return c.JSON(http.StatusConflict, dto.Err("User already a member"))
	}

	return c.JSON(http.StatusCreated, dto.OK(dto.SuccessResponse{Success: true}))
}

func (h *TeamHandler) ListProjects(c echo.Context) error {
	ctx := c.Request().Context()
	slug := c.Param("teamSlug")
	userID := mw.GetUserID(c)

	team, err := h.teamRepo.FindBySlug(ctx, slug)
	if err != nil {
		return c.JSON(http.StatusNotFound, dto.Err("Team not found"))
	}

	isMember, _ := h.teamRepo.IsMember(ctx, team.ID, userID)
	if !isMember {
		return c.JSON(http.StatusNotFound, dto.Err("Team not found"))
	}

	projects, err := h.projectRepo.FindByTeamID(ctx, team.ID)
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
	slug := c.Param("teamSlug")
	userID := mw.GetUserID(c)

	team, err := h.teamRepo.FindBySlug(ctx, slug)
	if err != nil {
		return c.JSON(http.StatusNotFound, dto.Err("Team not found"))
	}

	isMember, _ := h.teamRepo.IsMember(ctx, team.ID, userID)
	if !isMember {
		return c.JSON(http.StatusNotFound, dto.Err("Team not found"))
	}

	project, err := h.projectRepo.Create(ctx, team.ID, req.Name, req.Description, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to create project"))
	}

	return c.JSON(http.StatusCreated, dto.OK(project))
}
