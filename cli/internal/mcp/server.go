package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"github.com/thask/cli/internal/client"
)

var Version = "dev"

// Serve runs the MCP stdio loop. projectID is optional — when non-empty, the
// initialize response includes the user's recent mistakes so non–Claude-Code
// clients (Cursor, Codex CLI) get the same auto-injected context that
// Claude Code's SessionStart hook provides via `thask guide`.
func Serve(c *client.Client, projectID string) error {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024) // 10MB max
	encoder := json.NewEncoder(os.Stdout)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			encoder.Encode(ErrorResponse(nil, -32700, "Parse error"))
			continue
		}

		resp, ok := handleRequest(c, projectID, req)
		if ok {
			encoder.Encode(resp)
		}
	}

	return scanner.Err()
}

func handleRequest(c *client.Client, projectID string, req Request) (Response, bool) {
	switch req.Method {
	case "initialize":
		return SuccessResponse(req.ID, InitializeResult{
			ProtocolVersion: "2024-11-05",
			Capabilities:    Capabilities{Tools: &ToolsCapability{}},
			ServerInfo:      ServerInfo{Name: "thask", Version: Version},
			Instructions:    RenderInstructions(c, projectID),
		}), true

	case "notifications/initialized":
		return Response{}, false

	case "tools/list":
		return SuccessResponse(req.ID, ToolsListResult{Tools: AllTools()}), true

	case "tools/call":
		var params ToolCallParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return ErrorResponse(req.ID, -32602, "Invalid params"), true
		}

		result, err := HandleToolCall(c, params.Name, params.Arguments)
		if err != nil {
			return SuccessResponse(req.ID, ToolCallResult{
				Content: []ContentBlock{{Type: "text", Text: fmt.Sprintf("Error: %s", err.Error())}},
				IsError: true,
			}), true
		}

		var text string
		switch v := result.(type) {
		case json.RawMessage:
			// Pretty-print the JSON
			var buf any
			json.Unmarshal(v, &buf)
			out, _ := json.MarshalIndent(buf, "", "  ")
			text = string(out)
		default:
			out, _ := json.MarshalIndent(v, "", "  ")
			text = string(out)
		}

		return SuccessResponse(req.ID, ToolCallResult{
			Content: []ContentBlock{{Type: "text", Text: text}},
		}), true

	default:
		if req.ID == nil {
			return Response{}, false
		}
		return ErrorResponse(req.ID, -32601, "Method not found: "+req.Method), true
	}
}
