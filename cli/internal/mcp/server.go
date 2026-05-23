package mcp

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
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
	// Session UUID is generated once per MCP server lifetime; the audit_log
	// uses it to correlate writes made by the same agent conversation.
	sessionID := newSessionID()
	// Default header until clientInfo arrives via initialize. Always set so
	// backend can distinguish MCP from CLI even if init never lands.
	c.ClientHeader = fmt.Sprintf("thask-mcp/%s session=%s", Version, sessionID)

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

		resp, ok := handleRequest(c, projectID, sessionID, req)
		if ok {
			encoder.Encode(resp)
		}
	}

	return scanner.Err()
}

func newSessionID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func handleRequest(c *client.Client, projectID, sessionID string, req Request) (Response, bool) {
	switch req.Method {
	case "initialize":
		// Parse clientInfo so the X-Thask-Client header can carry the agent
		// model into audit_log. Failure here is non-fatal — the default
		// header set in Serve() remains.
		var p InitializeParams
		if len(req.Params) > 0 {
			_ = json.Unmarshal(req.Params, &p)
		}
		c.ClientHeader = buildMCPClientHeader(sessionID, p.ClientInfo)

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

// buildMCPClientHeader formats X-Thask-Client with the model/session info that
// the backend's audit_log indexes by. Falls back to "thask-mcp/<ver>" when
// clientInfo is missing.
func buildMCPClientHeader(sessionID string, info *InitializeClientInfo) string {
	if info == nil || info.Name == "" {
		return fmt.Sprintf("thask-mcp/%s session=%s", Version, sessionID)
	}
	// Encode the upstream agent identity (e.g. "claude-code/0.1.0") in the
	// model= param so the same MCP layer surfaces who is calling.
	model := info.Name
	if info.Version != "" {
		model = info.Name + "/" + info.Version
	}
	return fmt.Sprintf("thask-mcp/%s model=%s session=%s", Version, model, sessionID)
}
