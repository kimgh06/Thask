package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/thask/backend/internal/dto"
	mw "github.com/thask/backend/internal/middleware"
	"github.com/thask/backend/internal/model"
	"github.com/thask/backend/internal/repository"
	"github.com/thask/backend/internal/service"
)

type EdgeHandler struct {
	edgeRepo *repository.EdgeRepo
	hub      *service.Hub
}

func NewEdgeHandler(edgeRepo *repository.EdgeRepo, hub *service.Hub) *EdgeHandler {
	return &EdgeHandler{edgeRepo: edgeRepo, hub: hub}
}

func (h *EdgeHandler) List(c echo.Context) error {
	ctx := c.Request().Context()
	projectID := mw.ResolveProjectID(c)

	// V1 paginated path
	if c.Get(mw.ContextIsV1) == true {
		limitParam, _ := strconv.Atoi(c.QueryParam("limit"))
		limit := dto.ClampLimit(limitParam, 100)

		var afterTime *time.Time
		var afterID *string
		if cursor := c.QueryParam("after"); cursor != "" {
			t, id, err := dto.DecodeCursor(cursor)
			if err != nil {
				return c.JSON(http.StatusBadRequest, dto.V1Err(400, "Invalid cursor"))
			}
			afterTime = &t
			afterID = &id
		}

		edges, hasMore, err := h.edgeRepo.FindByProjectIDPaginated(ctx, projectID, limit, afterTime, afterID)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, dto.V1Err(500, "Failed to fetch edges"))
		}
		if edges == nil {
			edges = []model.Edge{}
		}

		var nextCursor *string
		if hasMore && len(edges) > 0 {
			last := edges[len(edges)-1]
			c := dto.EncodeCursor(last.CreatedAt, last.ID)
			nextCursor = &c
		}
		return c.JSON(http.StatusOK, dto.PaginatedResponse{
			Data:       edges,
			Pagination: dto.PaginationMeta{Limit: limit, HasMore: hasMore, NextCursor: nextCursor},
		})
	}

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
	projectID := mw.ResolveProjectID(c)

	if req.SourceID == req.TargetID {
		return c.JSON(http.StatusBadRequest, dto.Err("Self-referencing edges are not allowed"))
	}

	edgeType := model.EdgeType(req.EdgeType)
	if edgeType == "" {
		edgeType = model.EdgeTypeRelated
	}

	var edge *model.Edge
	var err error
	if req.SourcePort != "" || req.TargetPort != "" || req.Waypoints != nil {
		edge, err = h.edgeRepo.CreateWithRouting(ctx, projectID, req.SourceID, req.TargetID, edgeType, req.Label, req.SourcePort, req.TargetPort, req.Waypoints)
	} else {
		edge, err = h.edgeRepo.Create(ctx, projectID, req.SourceID, req.TargetID, edgeType, req.Label)
	}
	if err != nil {
		return c.JSON(http.StatusConflict, dto.Err("Edge already exists or invalid"))
	}

	h.hub.Publish(service.Event{Type: service.EventEdgeCreated, ProjectID: projectID, Data: edge, UserID: mw.GetUserID(c)})
	return c.JSON(http.StatusCreated, dto.OK(edge))
}

func (h *EdgeHandler) Update(c echo.Context) error {
	var req dto.UpdateEdgeRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err("Invalid request body"))
	}

	ctx := c.Request().Context()
	projectID := mw.ResolveProjectID(c)
	edgeID := c.Param("edgeId")

	// Verify edge belongs to this project
	if err := h.edgeRepo.VerifyProject(ctx, edgeID, projectID); err != nil {
		return c.JSON(http.StatusNotFound, dto.Err("Edge not found"))
	}

	var edgeType *model.EdgeType
	if req.EdgeType != nil {
		et := model.EdgeType(*req.EdgeType)
		edgeType = &et
	}

	edge, err := h.edgeRepo.Update(ctx, edgeID, edgeType, req.Label)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to update edge"))
	}

	// Update routing if provided
	if req.SourcePort != nil || req.TargetPort != nil || req.Waypoints != nil {
		edge, err = h.edgeRepo.UpdateRouting(ctx, edgeID, req.SourcePort, req.TargetPort, req.Waypoints)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, dto.Err("Failed to update edge routing"))
		}
	}

	h.hub.Publish(service.Event{Type: service.EventEdgeUpdated, ProjectID: edge.ProjectID, Data: edge, UserID: mw.GetUserID(c)})
	return c.JSON(http.StatusOK, dto.OK(edge))
}

func (h *EdgeHandler) Delete(c echo.Context) error {
	ctx := c.Request().Context()
	projectID := mw.ResolveProjectID(c)
	edgeID := c.Param("edgeId")

	if err := h.edgeRepo.DeleteScoped(ctx, edgeID, projectID); err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to delete edge"))
	}

	h.hub.Publish(service.Event{Type: service.EventEdgeDeleted, ProjectID: projectID, Data: edgeID, UserID: mw.GetUserID(c)})
	return c.JSON(http.StatusOK, dto.OK(dto.SuccessResponse{Success: true}))
}
