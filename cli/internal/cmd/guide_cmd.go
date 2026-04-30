package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thask/cli/internal/client"
	"github.com/thask/cli/internal/mcp"
)

var guideCmd = &cobra.Command{
	Use:   "guide",
	Short: "Print the AI agent interaction guide for Thask",
	Long: `Display the full guide that describes how AI agents should interact with Thask via MCP tools or CLI. Covers node/edge types, conventions, workflows, and pitfalls.

When a project is configured (--project, THASK_PROJECT, or thask config set project),
the output also includes a "Your Project Context" section: recent mistakes,
in-progress work, and blockers — so AI agents loading this guide get the user's
current state, not just static reference material.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		pid := resolveProject()
		// `guide` is in the skipAuth set so apiClient is not pre-initialized.
		// Build it lazily when config is sufficient — otherwise fall back to the
		// static guide so the command still works offline.
		var c *client.Client
		if pid != "" && cfg.URL != "" && cfg.Token != "" {
			c = client.New(cfg.URL, cfg.Token)
		}
		fmt.Print(mcp.RenderGuide(c, pid))
		return nil
	},
}
