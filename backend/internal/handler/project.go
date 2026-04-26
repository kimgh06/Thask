package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/thask/backend/internal/dto"
	mw "github.com/thask/backend/internal/middleware"
	"github.com/thask/backend/internal/model"
	"github.com/thask/backend/internal/repository"
	"github.com/thask/backend/internal/service"
)

type ProjectHandler struct {
	projectRepo *repository.ProjectRepo
	teamRepo    *repository.TeamRepo
	pmRepo      *repository.ProjectMemberRepo
	userRepo    *repository.UserRepo
}

func NewProjectHandler(projectRepo *repository.ProjectRepo, teamRepo *repository.TeamRepo, pmRepo *repository.ProjectMemberRepo, userRepo *repository.UserRepo) *ProjectHandler {
	return &ProjectHandler{projectRepo: projectRepo, teamRepo: teamRepo, pmRepo: pmRepo, userRepo: userRepo}
}

func (h *ProjectHandler) Get(c echo.Context) error {
	ctx := c.Request().Context()
	projectID := c.Param("projectId")

	project, err := h.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		return c.JSON(http.StatusNotFound, dto.Err("Project not found"))
	}

	return c.JSON(http.StatusOK, dto.OK(project))
}

func (h *ProjectHandler) Update(c echo.Context) error {
	var req dto.UpdateProjectRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err("Invalid request body"))
	}

	ctx := c.Request().Context()
	projectID := c.Param("projectId")

	project, err := h.projectRepo.Update(ctx, projectID, req.Name, req.Description)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to update project"))
	}

	return c.JSON(http.StatusOK, dto.OK(project))
}

func (h *ProjectHandler) Delete(c echo.Context) error {
	ctx := c.Request().Context()
	projectID := c.Param("projectId")

	if err := h.projectRepo.Delete(ctx, projectID); err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to delete project"))
	}

	return c.JSON(http.StatusOK, dto.OK(dto.SuccessResponse{Success: true}))
}

// --- Sharing ---

func (h *ProjectHandler) GetSharing(c echo.Context) error {
	ctx := c.Request().Context()
	projectID := c.Param("projectId")

	project, err := h.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		return c.JSON(http.StatusNotFound, dto.Err("Project not found"))
	}

	members, err := h.pmRepo.ListWithUsers(ctx, projectID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to fetch members"))
	}
	if members == nil {
		members = []model.ProjectMemberWithUser{}
	}

	var shareURL *string
	if project.ShareToken != nil {
		url := "/shared/" + *project.ShareToken
		shareURL = &url
	}

	return c.JSON(http.StatusOK, dto.OK(map[string]any{
		"linkSharing": project.LinkSharing,
		"shareUrl":    shareURL,
		"members":     members,
	}))
}

func (h *ProjectHandler) UpdateSharing(c echo.Context) error {
	var req dto.UpdateSharingRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err("Invalid request body"))
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err(err.Error()))
	}

	ctx := c.Request().Context()
	projectID := c.Param("projectId")

	var shareToken *string
	if req.LinkSharing != "off" {
		// Get existing token or generate new one
		project, err := h.projectRepo.FindByID(ctx, projectID)
		if err != nil {
			return c.JSON(http.StatusNotFound, dto.Err("Project not found"))
		}
		if project.ShareToken != nil {
			shareToken = project.ShareToken
		} else {
			token, err := service.GenerateToken()
			if err != nil {
				return c.JSON(http.StatusInternalServerError, dto.Err("Failed to generate share token"))
			}
			shareToken = &token
		}
	}
	// If "off", shareToken stays nil → clears the token intentionally.
	// Re-enabling sharing generates a new token, invalidating old links (security by design).

	updated, err := h.projectRepo.UpdateLinkSharing(ctx, projectID, req.LinkSharing, shareToken)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to update sharing"))
	}

	var shareURL *string
	if updated.ShareToken != nil {
		url := "/shared/" + *updated.ShareToken
		shareURL = &url
	}

	return c.JSON(http.StatusOK, dto.OK(map[string]any{
		"linkSharing": updated.LinkSharing,
		"shareUrl":    shareURL,
	}))
}

// --- Project Members ---

func (h *ProjectHandler) AddMember(c echo.Context) error {
	var req dto.AddProjectMemberRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err("Invalid request body"))
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err(err.Error()))
	}

	ctx := c.Request().Context()
	projectID := c.Param("projectId")

	user, err := h.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return c.JSON(http.StatusNotFound, dto.Err("User not found"))
	}

	role := model.ProjectRole(req.Role)
	if err := h.pmRepo.Add(ctx, projectID, user.ID, role); err != nil {
		return c.JSON(http.StatusConflict, dto.Err("User already a project member"))
	}

	return c.JSON(http.StatusCreated, dto.OK(dto.SuccessResponse{Success: true}))
}

func (h *ProjectHandler) UpdateMember(c echo.Context) error {
	var req dto.UpdateProjectMemberRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err("Invalid request body"))
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err(err.Error()))
	}

	ctx := c.Request().Context()
	projectID := c.Param("projectId")
	userID := c.Param("userId")

	if err := h.pmRepo.UpdateRole(ctx, projectID, userID, model.ProjectRole(req.Role)); err != nil {
		return c.JSON(http.StatusNotFound, dto.Err("Project member not found"))
	}

	return c.JSON(http.StatusOK, dto.OK(dto.SuccessResponse{Success: true}))
}

func (h *ProjectHandler) RemoveMember(c echo.Context) error {
	ctx := c.Request().Context()
	projectID := c.Param("projectId")
	userID := c.Param("userId")

	if err := h.pmRepo.Remove(ctx, projectID, userID); err != nil {
		return c.JSON(http.StatusNotFound, dto.Err("Project member not found"))
	}

	return c.JSON(http.StatusOK, dto.OK(dto.SuccessResponse{Success: true}))
}

// --- Shared (public, no auth) ---

func (h *ProjectHandler) SharedGet(c echo.Context) error {
	ctx := c.Request().Context()
	projectID := mw.GetProjectID(c)

	project, err := h.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		return c.JSON(http.StatusNotFound, dto.Err("Not found"))
	}

	graphUpdatedAt, err := h.projectRepo.GraphUpdatedAt(ctx, projectID)
	if err != nil {
		graphUpdatedAt = project.UpdatedAt
	}

	// Return limited info for public access
	return c.JSON(http.StatusOK, dto.OK(map[string]any{
		"id":             project.ID,
		"name":           project.Name,
		"description":    project.Description,
		"linkSharing":    project.LinkSharing,
		"graphUpdatedAt": graphUpdatedAt,
	}))
}

