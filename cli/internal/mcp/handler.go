package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/thask/cli/internal/client"
	"github.com/thask/cli/internal/scan"
)

type ToolHandler func(c *client.Client, args map[string]any) (any, error)

var handlers = map[string]ToolHandler{
	"thask.node.list":         handleNodeList,
	"thask.node.create":       handleNodeCreate,
	"thask.node.get":          handleNodeGet,
	"thask.node.update":       handleNodeUpdate,
	"thask.node.delete":       handleNodeDelete,
	"thask.node.batch_status": handleNodeBatchStatus,
	"thask.edge.list":         handleEdgeList,
	"thask.edge.create":       handleEdgeCreate,
	"thask.edge.delete":       handleEdgeDelete,
	"thask.graph.get":         handleGraphGet,
	"thask.graph.import":      handleGraphImport,
	"thask.impact.analyze":    handleImpactAnalyze,
	"thask.graph.layout":      handleGraphLayout,
	"thask.scan.run":          handleScanRun,
	"thask.graph.analyze":     handleGraphAnalyze,
	"thask.mistake.record":    handleMistakeRecord,
	"thask.guide":             handleGuide,

	// Provenance / human-in-the-loop tools (suggestion queue + verify).
	"thask.node.suggest_update": handleNodeSuggestUpdate,
	"thask.suggestions.list":    handleSuggestionsList,
	"thask.suggestions.decide":  handleSuggestionsDecide,
	"thask.node.verify":         handleNodeVerify,
}

func HandleToolCall(c *client.Client, name string, args json.RawMessage) (any, error) {
	handler, ok := handlers[name]
	if !ok {
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
	var params map[string]any
	if len(args) > 0 {
		if err := json.Unmarshal(args, &params); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
	}
	return handler(c, params)
}

func str(args map[string]any, key string) string {
	v, _ := args[key].(string)
	return v
}

func handleNodeList(c *client.Client, args map[string]any) (any, error) {
	pid := str(args, "projectId")
	path := "/api/projects/" + pid + "/nodes"
	var params []string
	if t := str(args, "type"); t != "" {
		params = append(params, "type="+t)
	}
	if s := str(args, "status"); s != "" {
		params = append(params, "status="+s)
	}
	if len(params) > 0 {
		path += "?" + strings.Join(params, "&")
	}
	return c.Get(path)
}

func handleNodeCreate(c *client.Client, args map[string]any) (any, error) {
	pid := str(args, "projectId")
	body := map[string]any{
		"type":  str(args, "type"),
		"title": str(args, "title"),
	}
	if v := str(args, "description"); v != "" {
		body["description"] = v
	}
	if v := str(args, "status"); v != "" {
		body["status"] = v
	}
	if v, ok := args["tags"]; ok {
		body["tags"] = v
	}
	if v, ok := args["positionX"]; ok {
		body["positionX"] = v
	}
	if v, ok := args["positionY"]; ok {
		body["positionY"] = v
	}
	return c.Post("/api/projects/"+pid+"/nodes", body)
}

func handleNodeGet(c *client.Client, args map[string]any) (any, error) {
	return c.Get("/api/projects/" + str(args, "projectId") + "/nodes/" + str(args, "nodeId"))
}

func handleNodeUpdate(c *client.Client, args map[string]any) (any, error) {
	pid := str(args, "projectId")
	nid := str(args, "nodeId")
	body := map[string]any{}
	for _, k := range []string{"title", "status", "type", "description"} {
		if v := str(args, k); v != "" {
			body[k] = v
		}
	}
	if v, ok := args["tags"]; ok {
		body["tags"] = v
	}
	return c.Patch("/api/projects/"+pid+"/nodes/"+nid, body)
}

func handleNodeDelete(c *client.Client, args map[string]any) (any, error) {
	return c.Delete("/api/projects/" + str(args, "projectId") + "/nodes/" + str(args, "nodeId"))
}

func handleNodeBatchStatus(c *client.Client, args map[string]any) (any, error) {
	pid := str(args, "projectId")
	return c.Patch("/api/projects/"+pid+"/nodes/batch-status", map[string]any{
		"ids":    args["ids"],
		"status": str(args, "status"),
	})
}

func handleEdgeList(c *client.Client, args map[string]any) (any, error) {
	return c.Get("/api/projects/" + str(args, "projectId") + "/edges")
}

func handleEdgeCreate(c *client.Client, args map[string]any) (any, error) {
	pid := str(args, "projectId")
	body := map[string]any{
		"sourceId": str(args, "sourceId"),
		"targetId": str(args, "targetId"),
	}
	if v := str(args, "edgeType"); v != "" {
		body["edgeType"] = v
	}
	if v := str(args, "label"); v != "" {
		body["label"] = v
	}
	return c.Post("/api/projects/"+pid+"/edges", body)
}

func handleEdgeDelete(c *client.Client, args map[string]any) (any, error) {
	return c.Delete("/api/projects/" + str(args, "projectId") + "/edges/" + str(args, "edgeId"))
}

func handleGraphGet(c *client.Client, args map[string]any) (any, error) {
	return c.Get("/api/projects/" + str(args, "projectId") + "/graph")
}

func handleGraphImport(c *client.Client, args map[string]any) (any, error) {
	pid := str(args, "projectId")
	return c.Post("/api/projects/"+pid+"/graph/import", map[string]any{
		"mode":  str(args, "mode"),
		"nodes": args["nodes"],
		"edges": args["edges"],
	})
}

func handleImpactAnalyze(c *client.Client, args map[string]any) (any, error) {
	return c.Get("/api/projects/" + str(args, "projectId") + "/impact?nodeId=" + str(args, "nodeId"))
}

func handleGraphLayout(c *client.Client, args map[string]any) (any, error) {
	pid := str(args, "projectId")
	body := map[string]any{}
	if algo := str(args, "algorithm"); algo != "" {
		body["algorithm"] = algo
	}
	return c.Post("/api/projects/"+pid+"/graph/layout", body)
}

func handleScanRun(c *client.Client, args map[string]any) (any, error) {
	pid := str(args, "projectId")
	path := filepath.Clean(str(args, "path"))

	// Validate path exists and is a directory
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %s", path)
	}

	maxFiles := 0
	if v, ok := args["maxFiles"].(float64); ok {
		maxFiles = int(v)
	}

	language := str(args, "language")
	if language == "auto" {
		language = ""
	}

	result, err := scan.Run(scan.ScanOptions{Path: path, MaxFiles: maxFiles, Language: language})
	if err != nil {
		return nil, err
	}

	_, err = c.Post("/api/projects/"+pid+"/graph/import", result)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"nodesCreated": len(result.Nodes),
		"edgesCreated": len(result.Edges),
	}, nil
}

func handleGraphAnalyze(c *client.Client, args map[string]any) (any, error) {
	return c.Get("/api/projects/" + str(args, "projectId") + "/graph/analyze")
}

func handleGuide(c *client.Client, args map[string]any) (any, error) {
	pid := str(args, "projectId")
	return map[string]string{"guide": RenderGuide(c, pid)}, nil
}

const mistakeGroupTitle = "실수 기록"

func handleMistakeRecord(c *client.Client, args map[string]any) (any, error) {
	pid := str(args, "projectId")
	title := str(args, "title")
	lesson := str(args, "lesson")
	if pid == "" || title == "" || lesson == "" {
		return nil, fmt.Errorf("projectId, title, and lesson are required")
	}

	groupID, err := findOrCreateMistakeGroup(c, pid)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve mistake group: %w", err)
	}

	var sb strings.Builder
	if cause := str(args, "cause"); cause != "" {
		sb.WriteString("**원인:** ")
		sb.WriteString(cause)
		sb.WriteString("\n\n")
	}
	if fix := str(args, "fix"); fix != "" {
		sb.WriteString("**수정:** ")
		sb.WriteString(fix)
		sb.WriteString("\n\n")
	}
	sb.WriteString("**교훈:** ")
	sb.WriteString(lesson)

	raw, err := c.Post("/api/projects/"+pid+"/nodes", map[string]any{
		"type":        "BUG",
		"title":       title,
		"description": sb.String(),
		"status":      "FAIL",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create mistake node: %w", err)
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(raw, &created); err != nil {
		return nil, fmt.Errorf("failed to parse created node: %w", err)
	}

	if _, err := c.Patch("/api/projects/"+pid+"/nodes/"+created.ID, map[string]any{
		"parentId": groupID,
	}); err != nil {
		return nil, fmt.Errorf("failed to attach mistake to group: %w", err)
	}

	return map[string]any{
		"id":      created.ID,
		"groupId": groupID,
		"title":   title,
		"message": "Mistake recorded under '" + mistakeGroupTitle + "'.",
	}, nil
}

func findOrCreateMistakeGroup(c *client.Client, pid string) (string, error) {
	raw, err := c.Get("/api/projects/" + pid + "/nodes?type=GROUP")
	if err != nil {
		return "", err
	}
	var groups []struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}
	if err := json.Unmarshal(raw, &groups); err != nil {
		return "", fmt.Errorf("failed to parse group list: %w", err)
	}
	for _, g := range groups {
		if g.Title == mistakeGroupTitle || g.Title == "Mistakes" || g.Title == "Mistake Log" {
			return g.ID, nil
		}
	}

	raw, err = c.Post("/api/projects/"+pid+"/nodes", map[string]any{
		"type":        "GROUP",
		"title":       mistakeGroupTitle,
		"description": "Agent mistakes recorded for self-reinforcing learning. Surfaced via thask.guide in future sessions.",
	})
	if err != nil {
		return "", err
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(raw, &created); err != nil {
		return "", fmt.Errorf("failed to parse created group: %w", err)
	}
	return created.ID, nil
}

// handleNodeSuggestUpdate posts to the suggestion queue rather than writing
// the node directly. Agent keys must use this path for description changes.
func handleNodeSuggestUpdate(c *client.Client, args map[string]any) (any, error) {
	pid := str(args, "projectId")
	nid := str(args, "nodeId")
	body := map[string]any{
		"proposedValue": str(args, "proposedValue"),
	}
	if v := str(args, "fieldName"); v != "" {
		body["fieldName"] = v
	}
	if v := str(args, "rationale"); v != "" {
		body["rationale"] = v
	}
	if v, ok := args["evidence"]; ok {
		body["evidence"] = v
	}
	return c.Post("/api/projects/"+pid+"/nodes/"+nid+"/suggestions", body)
}

func handleSuggestionsList(c *client.Client, args map[string]any) (any, error) {
	pid := str(args, "projectId")
	path := "/api/projects/" + pid + "/suggestions"
	if l, ok := args["limit"]; ok {
		path += fmt.Sprintf("?limit=%v", l)
	}
	return c.Get(path)
}

func handleSuggestionsDecide(c *client.Client, args map[string]any) (any, error) {
	pid := str(args, "projectId")
	sid := str(args, "suggestionId")
	body := map[string]any{"status": str(args, "status")}
	if v := str(args, "reason"); v != "" {
		body["reason"] = v
	}
	return c.Patch("/api/projects/"+pid+"/suggestions/"+sid, body)
}

func handleNodeVerify(c *client.Client, args map[string]any) (any, error) {
	pid := str(args, "projectId")
	nid := str(args, "nodeId")
	body := map[string]any{}
	if v := str(args, "commit"); v != "" {
		body["commit"] = v
	}
	return c.Post("/api/projects/"+pid+"/nodes/"+nid+"/verify", body)
}
