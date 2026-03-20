package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thask/cli/internal/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage CLI configuration",
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a config value (url, token, team, project)",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		c := config.Load()
		if err := c.Set(args[0], args[1]); err != nil {
			return err
		}
		fmt.Printf("Set %s\n", args[0])
		return nil
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := config.Load()
		fmt.Printf("url:     %s\n", c.URL)
		fmt.Printf("token:   %s\n", c.MaskedToken())
		fmt.Printf("team:    %s\n", c.Team)
		fmt.Printf("project: %s\n", c.Project)
		return nil
	},
}

func init() {
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configShowCmd)
}
