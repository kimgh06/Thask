package mcp

import (
	"encoding/json"
	"fmt"
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
	path := str(args, "path")
	maxFiles := 0
	if v, ok := args["maxFiles"].(float64); ok {
		maxFiles = int(v)
	}

	result, err := scan.Run(scan.ScanOptions{Path: path, MaxFiles: maxFiles})
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
