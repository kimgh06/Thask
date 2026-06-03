package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/thask/cli/internal/client"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Interactive setup wizard — config, connection test, CLAUDE.md patch",
	Long: `Walks through Thask setup in one command:

  1. Prompts for backend URL and API token (or keeps existing config)
  2. Tests the connection by calling /api/auth/me
  3. Lists available teams and projects so you can pick a default
  4. Patches CLAUDE.md in the current directory with Thask conventions for AI agents
  5. Prints the Claude Code plugin install command

Safe to re-run — it never overwrites a configured value without confirmation.`,
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	in := bufio.NewReader(os.Stdin)

	fmt.Println("Thask setup")
	fmt.Println("───────────")
	fmt.Println()

	// Step 1: URL
	defaultURL := cfg.URL
	if defaultURL == "" {
		defaultURL = "http://localhost:7244"
	}
	url := promptDefault(in, "Backend URL", defaultURL)
	cfg.URL = strings.TrimRight(url, "/")

	// Step 2: Token
	tokenHint := cfg.Token
	if tokenHint != "" {
		// Mask all but last 4
		mask := strings.Repeat("•", 8)
		if len(tokenHint) > 4 {
			tokenHint = mask + tokenHint[len(tokenHint)-4:]
		}
		fmt.Printf("Existing token: %s\n", tokenHint)
		if !promptYesNo(in, "Replace token?", false) {
			tokenHint = cfg.Token
		} else {
			tokenHint = ""
		}
	}
	if tokenHint == "" || cfg.Token == "" {
		fmt.Println("Create an API key at: " + cfg.URL + "/settings/api-keys")
		token := promptDefault(in, "API token", "")
		if token == "" {
			return fmt.Errorf("token is required")
		}
		cfg.Token = token
	}

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Step 3: Test connection
	c := client.New(cfg.URL, cfg.Token)
	apiClient = c
	fmt.Println()
	fmt.Print("Testing connection... ")
	raw, err := c.Get("/api/auth/me")
	if err != nil {
		fmt.Println("FAILED")
		return fmt.Errorf("connection test failed: %w", err)
	}
	var user struct {
		Email       string `json:"email"`
		DisplayName string `json:"displayName"`
	}
	_ = json.Unmarshal(raw, &user)
	fmt.Printf("OK — signed in as %s (%s)\n", user.DisplayName, user.Email)

	// Step 4: Pick project
	if err := pickDefaultProject(in, c); err != nil {
		fmt.Println("Skipping project selection:", err)
	}

	// Step 5: Patch CLAUDE.md
	if path, patched, err := patchClaudeMd(); err != nil {
		fmt.Println("CLAUDE.md not patched:", err)
	} else if patched {
		fmt.Printf("Updated %s with Thask agent conventions\n", path)
	} else {
		fmt.Printf("%s already contains Thask section — left untouched\n", path)
	}

	// Step 6: Auto-patch Cursor / Codex MCP configs if present
	for _, p := range patchAllMCPClients() {
		fmt.Println(p)
	}

	// Step 7: Plugin install hint (Claude Code only)
	fmt.Println()
	fmt.Println("For Claude Code, also install the plugin to enable SessionStart context:")
	fmt.Println("  /plugin marketplace add kimgh06/Thask")
	fmt.Println("  /plugin install thask@thask")
	fmt.Println()
	fmt.Println("Done.")
	fmt.Println("  • `thask doctor` — verify the full setup at a glance")
	fmt.Println("  • `thask guide`  — dynamic agent guide for this project")
	return nil
}

// patchAllMCPClients detects installed AI coding tool configs (Cursor, Codex)
// and adds a `thask` MCP server entry to each. Files that don't exist are
// skipped silently. Returns human-readable status lines.
func patchAllMCPClients() []string {
	var lines []string
	home, _ := os.UserHomeDir()

	// Cursor: ~/.cursor/mcp.json (global) and ./.cursor/mcp.json (project)
	for _, p := range []string{filepath.Join(home, ".cursor", "mcp.json"), filepath.Join(".cursor", "mcp.json")} {
		if msg := patchCursorMCP(p); msg != "" {
			lines = append(lines, msg)
		}
	}

	// Codex: ~/.codex/config.toml
	if msg := patchCodexMCP(filepath.Join(home, ".codex", "config.toml")); msg != "" {
		lines = append(lines, msg)
	}

	return lines
}

func patchCursorMCP(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var cfg map[string]any
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return fmt.Sprintf("Cursor config %s parse failed: %v — skipped", path, err)
	}
	servers, _ := cfg["mcpServers"].(map[string]any)
	if servers == nil {
		servers = map[string]any{}
		cfg["mcpServers"] = servers
	}
	if _, exists := servers["thask"]; exists {
		return fmt.Sprintf("Cursor %s already has thask — left untouched", path)
	}
	servers["thask"] = map[string]any{
		"command": "thask",
		"args":    []string{"mcp", "serve"},
	}
	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Sprintf("Cursor config %s marshal failed: %v — skipped", path, err)
	}
	if err := os.WriteFile(path, out, 0644); err != nil {
		return fmt.Sprintf("Cursor config %s write failed: %v", path, err)
	}
	return fmt.Sprintf("Patched Cursor MCP config: %s", path)
}

func patchCodexMCP(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	if strings.Contains(string(raw), "[mcp_servers.thask]") {
		return fmt.Sprintf("Codex %s already has thask — left untouched", path)
	}
	section := "\n[mcp_servers.thask]\ncommand = \"thask\"\nargs = [\"mcp\", \"serve\"]\n"
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Sprintf("Codex config %s open failed: %v", path, err)
	}
	defer f.Close()
	if _, err := f.WriteString(section); err != nil {
		return fmt.Sprintf("Codex config %s write failed: %v", path, err)
	}
	return fmt.Sprintf("Patched Codex MCP config: %s", path)
}

func pickDefaultProject(in *bufio.Reader, c *client.Client) error {
	team := cfg.Team
	if team == "" {
		// Auto-pick if exactly one team; otherwise prompt to set later.
		raw, err := c.Get("/api/teams")
		if err != nil {
			return err
		}
		var teams []struct {
			Slug string `json:"slug"`
			Name string `json:"name"`
		}
		if err := json.Unmarshal(raw, &teams); err != nil {
			return err
		}
		switch len(teams) {
		case 0:
			fmt.Println("No teams found. Create one in the web UI, then re-run `thask init`.")
			return nil
		case 1:
			team = teams[0].Slug
			cfg.Team = team
			_ = cfg.Save()
			fmt.Printf("Default team set to %s\n", teams[0].Name)
		default:
			fmt.Println()
			fmt.Println("Multiple teams found:")
			for i, t := range teams {
				fmt.Printf("  %d) %s  (%s)\n", i+1, t.Name, t.Slug)
			}
			choice := promptDefault(in, "Pick team number", "")
			var idx int
			if _, err := fmt.Sscanf(choice, "%d", &idx); err != nil || idx < 1 || idx > len(teams) {
				return fmt.Errorf("invalid team choice")
			}
			team = teams[idx-1].Slug
			cfg.Team = team
			_ = cfg.Save()
		}
	}

	raw, err := c.Get("/api/teams/" + team + "/projects")
	if err != nil {
		return err
	}
	var projects []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(raw, &projects); err != nil {
		return err
	}
	if len(projects) == 0 {
		fmt.Println("No projects yet. Create one in the web UI, then run `thask config set project <id>`.")
		return nil
	}
	fmt.Println()
	fmt.Println("Available projects:")
	for i, p := range projects {
		marker := " "
		if p.ID == cfg.Project {
			marker = "*"
		}
		fmt.Printf("  %s %d) %s  %s\n", marker, i+1, p.Name, p.ID)
	}
	choice := promptDefault(in, "Pick number (or blank to keep current)", "")
	if choice == "" {
		return nil
	}
	var idx int
	if _, err := fmt.Sscanf(choice, "%d", &idx); err != nil || idx < 1 || idx > len(projects) {
		fmt.Println("Invalid choice — keeping existing.")
		return nil
	}
	cfg.Project = projects[idx-1].ID
	if err := cfg.Save(); err != nil {
		return err
	}
	fmt.Printf("Default project set to %s\n", projects[idx-1].Name)
	return nil
}

const claudeMdMarker = "<!-- thask:agent-conventions -->"

const claudeMdSection = `

## Thask conventions for AI agents <!-- thask:agent-conventions -->

This project uses [Thask](https://thask.kimgh06.com) as the dependency-graph context layer.

**At session start:** call ` + "`mcp__thask__thask_guide`" + ` (or run ` + "`thask guide`" + `) to load the static reference plus this project's recent mistakes, in-progress work, and blockers.

**Whenever you slip up:** call ` + "`mcp__thask__thask_mistake_record`" + ` (or ` + "`thask mistake record`" + `) so the same error surfaces in next session's guide. Triggers:
- The user corrects you ("그게 아니고…", "왜 X 했냐")
- You misuse a CLI flag, MCP tool, or assume a command's behavior
- You repeat any item shown under "Recent mistakes — DO NOT repeat" in ` + "`thask guide`" + ` output
`

func patchClaudeMd() (string, bool, error) {
	path := "CLAUDE.md"
	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return path, false, err
	}
	if strings.Contains(string(existing), claudeMdMarker) {
		return path, false, nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil && filepath.Dir(path) != "." {
		return path, false, err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return path, false, err
	}
	defer f.Close()
	if _, err := f.WriteString(claudeMdSection); err != nil {
		return path, false, err
	}
	return path, true, nil
}

func promptDefault(in *bufio.Reader, label, def string) string {
	if def != "" {
		fmt.Printf("%s [%s]: ", label, def)
	} else {
		fmt.Printf("%s: ", label)
	}
	line, _ := in.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return def
	}
	return line
}

func promptYesNo(in *bufio.Reader, label string, def bool) bool {
	suffix := "[y/N]"
	if def {
		suffix = "[Y/n]"
	}
	fmt.Printf("%s %s: ", label, suffix)
	line, _ := in.ReadString('\n')
	line = strings.TrimSpace(strings.ToLower(line))
	if line == "" {
		return def
	}
	return line == "y" || line == "yes"
}
