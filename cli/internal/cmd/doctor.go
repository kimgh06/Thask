package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/thask/cli/internal/client"
	"github.com/thask/cli/internal/config"
)

// doctorCmd diagnoses the CLI/MCP setup so users hit a single command instead
// of correlating 401s, "tools not visible", and silent MCP failures across
// shell history. Each check is reported with ✓/⚠/✗ and a hint when failing.
var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose CLI, MCP, and server connectivity in one shot",
	Long: `Runs a sequence of checks against your local config, the configured
backend, and any installed AI-tool MCP integrations. Exit codes:

  0  all critical checks passed (warnings OK)
  1  one or more critical checks failed

Use this when "thask node list" fails and you cannot tell whether the URL,
token, server, or migrations are at fault.`,
	RunE: runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

type checkResult struct {
	name    string
	status  string // "ok" | "warn" | "fail"
	detail  string
	hint    string
}

const (
	iconOK   = "✓"
	iconWarn = "⚠"
	iconFail = "✗"
)

func runDoctor(cmd *cobra.Command, _ []string) error {
	// cfg is normally populated by root PersistentPreRunE, but the "doctor"
	// command is in the skipAuth path so cfg may still be set — defensive
	// re-load to make this command standalone.
	c := cfg
	if c == nil {
		c = config.Load()
	}

	var results []checkResult
	push := func(r checkResult) { results = append(results, r) }

	// 1. Binary
	push(checkResult{
		name:   "Binary version",
		status: "ok",
		detail: fmt.Sprintf("thask %s (%s)", Version, Commit),
	})

	// 2. Config file
	home, _ := os.UserHomeDir()
	cfgPath := filepath.Join(home, ".thask", "config.json")
	if st, err := os.Stat(cfgPath); err == nil {
		push(checkResult{name: "Config file", status: "ok", detail: fmt.Sprintf("%s (mode %o)", cfgPath, st.Mode().Perm())})
	} else {
		push(checkResult{name: "Config file", status: "warn", detail: "not found", hint: "Run `thask init` or `thask login` to create one"})
	}

	// 3. URL set
	if c.URL == "" {
		push(checkResult{name: "URL", status: "fail", detail: "unset", hint: "thask config set url <url>"})
		return finalize(results)
	}
	push(checkResult{name: "URL", status: "ok", detail: c.URL})

	// 4. URL reachable + health
	health, err := fetchHealth(c.URL)
	if err != nil {
		push(checkResult{
			name:   "Server reachable",
			status: "fail",
			detail: err.Error(),
			hint:   "Is the server running? Check `docker compose ps` or your hosted instance",
		})
		return finalize(results)
	}
	push(checkResult{
		name:   "Server reachable",
		status: "ok",
		detail: fmt.Sprintf("backend %s, uptime %s", health.Version, health.Uptime),
	})
	if health.DB == "ok" {
		push(checkResult{
			name:   "Server DB",
			status: "ok",
			detail: fmt.Sprintf("migrations applied: %d (latest version %d)", health.MigrationsApplied, health.MigrationVersion),
		})
	} else {
		push(checkResult{
			name:   "Server DB",
			status: "fail",
			detail: health.DBError,
			hint:   "Check the backend logs and DATABASE_URL",
		})
	}

	// 5. Token set
	if c.Token == "" {
		push(checkResult{name: "Token", status: "fail", detail: "unset", hint: "thask login"})
		return finalize(results)
	}
	push(checkResult{name: "Token", status: "ok", detail: c.MaskedToken()})

	// 6+7. Token valid + permissions (via /api/auth/me + /api/auth/api-keys)
	api := client.New(c.URL, c.Token)
	api.ClientHeader = "thask-cli/" + Version
	if me, err := fetchMe(api); err != nil {
		push(checkResult{name: "Token valid", status: "fail", detail: err.Error(), hint: "Token may be revoked — run `thask login`"})
	} else {
		push(checkResult{name: "Token valid", status: "ok", detail: fmt.Sprintf("%s <%s>", me.DisplayName, me.Email)})
		if perms := fetchPermissions(api, c.Token); perms != "" {
			push(checkResult{name: "Token permissions", status: "ok", detail: perms})
		}
	}

	// 8. Default team
	if c.Team == "" {
		push(checkResult{name: "Default team", status: "warn", detail: "unset", hint: "thask config set team <slug>"})
	} else {
		push(checkResult{name: "Default team", status: "ok", detail: c.Team})
	}

	// 9. Default project
	if c.Project == "" {
		push(checkResult{name: "Default project", status: "warn", detail: "unset", hint: "thask config set project <id>"})
	} else {
		push(checkResult{name: "Default project", status: "ok", detail: c.Project})
	}

	// 10. MCP self-test
	if path, err := exec.LookPath("thask"); err == nil {
		push(checkResult{name: "MCP binary on PATH", status: "ok", detail: path})
	} else {
		push(checkResult{name: "MCP binary on PATH", status: "warn", detail: "`thask` not in PATH", hint: "Run `thask install` or symlink the binary into PATH"})
	}
	for _, line := range detectMCPConfigs() {
		push(line)
	}

	return finalize(results)
}

func finalize(results []checkResult) error {
	var crit, warn int
	for _, r := range results {
		var icon string
		switch r.status {
		case "ok":
			icon = iconOK
		case "warn":
			icon = iconWarn
			warn++
		case "fail":
			icon = iconFail
			crit++
		}
		fmt.Printf("  %s %-26s %s\n", icon, r.name, r.detail)
		if r.hint != "" {
			fmt.Printf("       → %s\n", r.hint)
		}
	}
	fmt.Println()
	switch {
	case crit > 0:
		fmt.Printf("%d critical, %d warning(s). Fix the items marked ✗.\n", crit, warn)
		return fmt.Errorf("doctor: %d critical issue(s)", crit)
	case warn > 0:
		fmt.Printf("All critical checks passed. %d warning(s).\n", warn)
	default:
		fmt.Println("All checks passed.")
	}
	return nil
}

// fetchHealth talks to /api/health (falls back to /health) so the doctor can
// surface backend version + migration state without needing auth.
type healthDTO struct {
	Status            string `json:"status"`
	Version           string `json:"version"`
	DB                string `json:"db"`
	DBError           string `json:"dbError"`
	MigrationVersion  int    `json:"migrationVersion"`
	MigrationsApplied int    `json:"migrationsApplied"`
	Uptime            string `json:"uptime"`
}

func fetchHealth(baseURL string) (*healthDTO, error) {
	httpc := &http.Client{Timeout: 5 * time.Second}
	for _, p := range []string{"/api/health", "/health"} {
		req, err := http.NewRequest("GET", strings.TrimRight(baseURL, "/")+p, nil)
		if err != nil {
			continue
		}
		resp, err := httpc.Do(req)
		if err != nil {
			return nil, err
		}
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("read /api/health body: %w", readErr)
		}
		var h healthDTO
		if json.Unmarshal(body, &h) == nil && h.Status != "" {
			return &h, nil
		}
		// Legacy endpoint that only returned `{"status":"ok"}` with no
		// version/db fields — synthesize a minimal struct so the doctor can
		// still surface "server up" without lying about DB or migrations.
		if resp.StatusCode == http.StatusOK {
			return &healthDTO{Status: "ok", Version: "unknown", DB: "ok"}, nil
		}
	}
	return nil, fmt.Errorf("HTTP request to %s failed", baseURL)
}

type meDTO struct {
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
}

func fetchMe(c *client.Client) (*meDTO, error) {
	raw, err := c.Get("/api/auth/me")
	if err != nil {
		return nil, err
	}
	var m meDTO
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	if m.Email == "" {
		return nil, fmt.Errorf("response missing email field")
	}
	return &m, nil
}

// fetchPermissions enumerates the current account's API keys and returns a
// short permission summary for the one matching the in-use token (by prefix).
// Best-effort: returns "" on any failure rather than blocking the report.
func fetchPermissions(c *client.Client, token string) string {
	raw, err := c.Get("/api/auth/api-keys")
	if err != nil {
		return ""
	}
	var keys []struct {
		Kind        string          `json:"kind"`
		Prefix      string          `json:"prefix"`
		Permissions map[string]bool `json:"permissions"`
	}
	if err := json.Unmarshal(raw, &keys); err != nil {
		return ""
	}
	for _, k := range keys {
		if k.Prefix != "" && strings.HasPrefix(token, k.Prefix) {
			var flags []string
			for name, v := range k.Permissions {
				if v {
					flags = append(flags, name)
				}
			}
			if len(flags) == 0 {
				return fmt.Sprintf("kind=%s, no permissions", k.Kind)
			}
			return fmt.Sprintf("kind=%s, %s", k.Kind, strings.Join(flags, ","))
		}
	}
	return ""
}

// detectMCPConfigs reports whether known AI-tool config files include a
// `thask` MCP server entry so users see where they are wired in.
func detectMCPConfigs() []checkResult {
	var out []checkResult
	home, _ := os.UserHomeDir()
	targets := []struct {
		label string
		path  string
		needle string
	}{
		{"Claude Code MCP", filepath.Join(home, ".claude.json"), "\"thask\""},
		{"Cursor MCP (global)", filepath.Join(home, ".cursor", "mcp.json"), "\"thask\""},
		{"Codex MCP", filepath.Join(home, ".codex", "config.toml"), "mcp_servers.thask"},
	}
	for _, t := range targets {
		raw, err := os.ReadFile(t.path)
		if err != nil {
			out = append(out, checkResult{name: t.label, status: "warn", detail: "not configured", hint: "Run `thask init` to patch it"})
			continue
		}
		if strings.Contains(string(raw), t.needle) {
			out = append(out, checkResult{name: t.label, status: "ok", detail: t.path})
		} else {
			out = append(out, checkResult{name: t.label, status: "warn", detail: "no thask entry", hint: "Run `thask init` to add it"})
		}
	}
	return out
}
