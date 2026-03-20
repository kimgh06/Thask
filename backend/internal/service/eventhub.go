package service

import "sync"

type EventType string

const (
	EventNodeCreated EventType = "node.created"
	EventNodeUpdated EventType = "node.updated"
	EventNodeDeleted EventType = "node.deleted"
	EventEdgeCreated EventType = "edge.created"
	EventEdgeUpdated EventType = "edge.updated"
	EventEdgeDeleted EventType = "edge.deleted"
	EventGraphLayout EventType = "graph.layout"
	EventGraphImport EventType = "graph.import"
)

type Event struct {
	Type      EventType `json:"type"`
	ProjectID string    `json:"projectId"`
	Data      any       `json:"data,omitempty"`
	UserID    string    `json:"userId"`
}

type Hub struct {
	mu          sync.RWMutex
	subscribers map[string]map[chan Event]bool
}

func NewHub() *Hub {
	return &Hub{
		subscribers: make(map[string]map[chan Event]bool),
	}
}

func (h *Hub) Subscribe(projectID string) chan Event {
	h.mu.Lock()
	defer h.mu.Unlock()

	ch := make(chan Event, 32)
	if h.subscribers[projectID] == nil {
		h.subscribers[projectID] = make(map[chan Event]bool)
	}
	h.subscribers[projectID][ch] = true
	return ch
}

func (h *Hub) Unsubscribe(projectID string, ch chan Event) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if subs, ok := h.subscribers[projectID]; ok {
		delete(subs, ch)
		close(ch)
		if len(subs) == 0 {
			delete(h.subscribers, projectID)
		}
	}
}

func (h *Hub) Publish(event Event) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	subs, ok := h.subscribers[event.ProjectID]
	if !ok {
		return
	}

	for ch := range subs {
		select {
		case ch <- event:
		default:
			// Channel full, skip (slow consumer)
		}
	}
}
