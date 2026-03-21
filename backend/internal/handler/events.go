package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	mw "github.com/thask/backend/internal/middleware"
	"github.com/thask/backend/internal/service"
)

type EventHandler struct {
	hub *service.Hub
}

func NewEventHandler(hub *service.Hub) *EventHandler {
	return &EventHandler{hub: hub}
}

func (h *EventHandler) Stream(c echo.Context) error {
	projectID := mw.GetProjectID(c)

	w := c.Response()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // nginx proxy support
	w.WriteHeader(http.StatusOK)

	ch := h.hub.Subscribe(projectID)
	defer h.hub.Unsubscribe(projectID, ch)

	// Initial connection event
	fmt.Fprintf(w, "event: connected\ndata: {}\n\n")
	w.Flush()

	ctx := c.Request().Context()
	heartbeat := time.NewTicker(30 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case event, ok := <-ch:
			if !ok {
				return nil
			}
			data, err := json.Marshal(event)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, data)
			w.Flush()
		case <-heartbeat.C:
			fmt.Fprintf(w, ": heartbeat\n\n")
			w.Flush()
		case <-ctx.Done():
			return nil
		}
	}
}
