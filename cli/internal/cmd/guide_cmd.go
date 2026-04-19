package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thask/cli/internal/mcp"
)

var guideCmd = &cobra.Command{
	Use:   "guide",
	Short: "Print the AI agent interaction guide for Thask",
	Long:  "Display the full guide that describes how AI agents should interact with Thask via MCP tools or CLI. Covers node/edge types, conventions, workflows, and pitfalls.",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(mcp.GuideText)
		return nil
	},
}
