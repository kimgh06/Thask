package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thask/cli/internal/client"
	cfgpkg "github.com/thask/cli/internal/config"
	"github.com/thask/cli/internal/mcp"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "MCP server for AI agent integration",
}

var mcpServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start MCP server on stdio (for Claude Code integration)",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := cfgpkg.Load()
		if urlFlag != "" {
			c.URL = urlFlag
		}
		if tokenFlag != "" {
			c.Token = tokenFlag
		}
		if projectFlag != "" {
			c.Project = projectFlag
		}

		if err := c.Validate(); err != nil {
			// MCP runs from inside Claude Code / Cursor where pasting a token
			// is awkward; point the user at the browser flow first.
			return fmt.Errorf("%w — run `thask login` to authenticate, then restart your MCP client", err)
		}

		apiC := client.New(c.URL, c.Token)
		return mcp.Serve(apiC, c.Project)
	},
}

func init() {
	mcpCmd.AddCommand(mcpServeCmd)
}
