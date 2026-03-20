package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thask/cli/internal/output"
)

var impactCmd = &cobra.Command{
	Use:     "impact",
	Aliases: []string{"im"},
	Short:   "Run impact analysis",
	RunE: func(cmd *cobra.Command, args []string) error {
		pid := resolveProject()
		if pid == "" {
			return fmt.Errorf("--project or THASK_PROJECT required")
		}
		nodeID, _ := cmd.Flags().GetString("node")
		if nodeID == "" {
			return fmt.Errorf("--node required")
		}
		data, err := apiClient.Get("/api/projects/" + pid + "/impact?nodeId=" + nodeID)
		if err != nil {
			return err
		}
		output.JSON(data)
		return nil
	},
}

func init() {
	impactCmd.Flags().String("node", "", "Node ID to analyze (required)")
	_ = impactCmd.MarkFlagRequired("node")
}
