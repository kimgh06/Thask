package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/thask/backend/internal/dto"
	"github.com/thask/backend/internal/repository"
)

type ProjectHandler struct {
	projectRepo *repository.ProjectRepo
	teamRepo    *repository.TeamRepo
}

func NewProjectHandler(projectRepo *repository.ProjectRepo, teamRepo *repository.TeamRepo) *ProjectHandler {
	return &ProjectHandler{projectRepo: projectRepo, teamRepo: teamRepo}
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
