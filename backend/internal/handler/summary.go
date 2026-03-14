package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/thask/backend/internal/dto"
	mw "github.com/thask/backend/internal/middleware"
	"github.com/thask/backend/internal/repository"
)

type SummaryHandler struct {
	teamRepo    *repository.TeamRepo
	projectRepo *repository.ProjectRepo
}

func NewSummaryHandler(teamRepo *repository.TeamRepo, projectRepo *repository.ProjectRepo) *SummaryHandler {
	return &SummaryHandler{teamRepo: teamRepo, projectRepo: projectRepo}
}

func (h *SummaryHandler) Get(c echo.Context) error {
	ctx := c.Request().Context()
	userID := mw.GetUserID(c)

	teams, err := h.teamRepo.FindByUserID(ctx, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to fetch teams"))
	}

	if len(teams) == 0 {
		return c.JSON(http.StatusOK, dto.OK(map[string]int{"totalProjects": 0, "totalTeams": 0}))
	}

	teamIDs := make([]string, len(teams))
	for i, t := range teams {
		teamIDs[i] = t.ID
	}

	projects, err := h.projectRepo.FindByTeamIDs(ctx, teamIDs)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to fetch projects"))
	}

	return c.JSON(http.StatusOK, dto.OK(map[string]int{
		"totalProjects": len(projects),
		"totalTeams":    len(teams),
	}))
}
