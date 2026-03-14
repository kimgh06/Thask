package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/thask/backend/internal/dto"
	"github.com/thask/backend/internal/model"
	"github.com/thask/backend/internal/repository"
)

type EdgeHandler struct {
	edgeRepo *repository.EdgeRepo
}

func NewEdgeHandler(edgeRepo *repository.EdgeRepo) *EdgeHandler {
	return &EdgeHandler{edgeRepo: edgeRepo}
}

func (h *EdgeHandler) List(c echo.Context) error {
	ctx := c.Request().Context()
	projectID := c.Param("projectId")

	edges, err := h.edgeRepo.FindByProjectID(ctx, projectID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to fetch edges"))
	}
	if edges == nil {
		edges = []model.Edge{}
	}

	return c.JSON(http.StatusOK, dto.OK(edges))
}

func (h *EdgeHandler) Create(c echo.Context) error {
	var req dto.CreateEdgeRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err("Invalid request body"))
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err(err.Error()))
	}

	ctx := c.Request().Context()
	projectID := c.Param("projectId")

	if req.SourceID == req.TargetID {
		return c.JSON(http.StatusBadRequest, dto.Err("Self-referencing edges are not allowed"))
	}

	edgeType := model.EdgeType(req.EdgeType)
	if edgeType == "" {
		edgeType = model.EdgeTypeRelated
	}

	edge, err := h.edgeRepo.Create(ctx, projectID, req.SourceID, req.TargetID, edgeType, req.Label)
	if err != nil {
		return c.JSON(http.StatusConflict, dto.Err("Edge already exists or invalid"))
	}

	return c.JSON(http.StatusCreated, dto.OK(edge))
}

func (h *EdgeHandler) Update(c echo.Context) error {
	var req dto.UpdateEdgeRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err("Invalid request body"))
	}

	ctx := c.Request().Context()
	edgeID := c.Param("edgeId")

	var edgeType *model.EdgeType
	if req.EdgeType != nil {
		et := model.EdgeType(*req.EdgeType)
		edgeType = &et
	}

	edge, err := h.edgeRepo.Update(ctx, edgeID, edgeType, req.Label)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to update edge"))
	}

	return c.JSON(http.StatusOK, dto.OK(edge))
}

func (h *EdgeHandler) Delete(c echo.Context) error {
	ctx := c.Request().Context()
	edgeID := c.Param("edgeId")

	if err := h.edgeRepo.Delete(ctx, edgeID); err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to delete edge"))
	}

	return c.JSON(http.StatusOK, dto.OK(dto.SuccessResponse{Success: true}))
}
