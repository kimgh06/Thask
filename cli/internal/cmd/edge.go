package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thask/cli/internal/output"
)

var edgeCmd = &cobra.Command{
	Use:     "edge",
	Aliases: []string{"e"},
	Short:   "Edge operations",
}

var edgeListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List edges in a project",
	RunE: func(cmd *cobra.Command, args []string) error {
		pid := resolveProject()
		if pid == "" {
			return fmt.Errorf("--project or THASK_PROJECT required")
		}
		data, err := apiClient.Get("/api/projects/" + pid + "/edges")
		if err != nil {
			return err
		}
		switch getFormat() {
		case output.FormatTable:
			var edges []struct {
				ID       string  `json:"id"`
				SourceID string  `json:"sourceId"`
				TargetID string  `json:"targetId"`
				EdgeType string  `json:"edgeType"`
				Label    *string `json:"label"`
			}
			json.Unmarshal(data, &edges)
			rows := make([][]string, len(edges))
			for i, e := range edges {
				label := ""
				if e.Label != nil {
					label = *e.Label
				}
				rows[i] = []string{e.ID[:8], e.SourceID[:8], e.TargetID[:8], e.EdgeType, label}
			}
			output.Table([]string{"ID", "SOURCE", "TARGET", "TYPE", "LABEL"}, rows)
		case output.FormatQuiet:
			var edges []struct{ ID string `json:"id"` }
			json.Unmarshal(data, &edges)
			ids := make([]string, len(edges))
			for i, e := range edges {
				ids[i] = e.ID
			}
			output.Quiet(ids)
		default:
			output.JSON(data)
		}
		return nil
	},
}

var edgeCreateCmd = &cobra.Command{
	Use:     "create",
	Aliases: []string{"c"},
	Short:   "Create an edge",
	RunE: func(cmd *cobra.Command, args []string) error {
		pid := resolveProject()
		if pid == "" {
			return fmt.Errorf("--project or THASK_PROJECT required")
		}
		source, _ := cmd.Flags().GetString("source")
		target, _ := cmd.Flags().GetString("target")
		edgeType, _ := cmd.Flags().GetString("type")
		label, _ := cmd.Flags().GetString("label")

		body := map[string]any{
			"sourceId": source,
			"targetId": target,
		}
		if edgeType != "" {
			body["edgeType"] = edgeType
		}
		if label != "" {
			body["label"] = label
		}

		data, err := apiClient.Post("/api/projects/"+pid+"/edges", body)
		if err != nil {
			return err
		}
		switch getFormat() {
		case output.FormatQuiet:
			var edge struct{ ID string `json:"id"` }
			json.Unmarshal(data, &edge)
			fmt.Println(edge.ID)
		default:
			output.JSON(data)
		}
		runLayoutIfFlagged(cmd)
		return nil
	},
}

var edgeUpdateCmd = &cobra.Command{
	Use:   "update <edgeId>",
	Short: "Update an edge",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pid := resolveProject()
		if pid == "" {
			return fmt.Errorf("--project or THASK_PROJECT required")
		}
		body := map[string]any{}
		if cmd.Flags().Changed("type") {
			v, _ := cmd.Flags().GetString("type")
			body["edgeType"] = v
		}
		if cmd.Flags().Changed("label") {
			v, _ := cmd.Flags().GetString("label")
			body["label"] = v
		}
		data, err := apiClient.Patch("/api/projects/"+pid+"/edges/"+args[0], body)
		if err != nil {
			return err
		}
		output.JSON(data)
		return nil
	},
}

var edgeDeleteCmd = &cobra.Command{
	Use:     "delete <edgeId>",
	Aliases: []string{"d", "rm"},
	Short:   "Delete an edge",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pid := resolveProject()
		if pid == "" {
			return fmt.Errorf("--project or THASK_PROJECT required")
		}
		_, err := apiClient.Delete("/api/projects/" + pid + "/edges/" + args[0])
		if err != nil {
			return err
		}
		fmt.Println("Deleted")
		runLayoutIfFlagged(cmd)
		return nil
	},
}

func init() {
	edgeCreateCmd.Flags().String("source", "", "Source node ID (required)")
	edgeCreateCmd.Flags().String("target", "", "Target node ID (required)")
	edgeCreateCmd.Flags().String("type", "", "Edge type: depends_on, blocks, related, parent_child, triggers")
	edgeCreateCmd.Flags().String("label", "", "Edge label")
	edgeCreateCmd.Flags().Bool("layout", false, "Run auto-layout after creation")
	_ = edgeCreateCmd.MarkFlagRequired("source")
	_ = edgeCreateCmd.MarkFlagRequired("target")

	edgeDeleteCmd.Flags().Bool("layout", false, "Run auto-layout after deletion")

	edgeUpdateCmd.Flags().String("type", "", "New edge type")
	edgeUpdateCmd.Flags().String("label", "", "New label")

	edgeCmd.AddCommand(edgeListCmd)
	edgeCmd.AddCommand(edgeCreateCmd)
	edgeCmd.AddCommand(edgeUpdateCmd)
	edgeCmd.AddCommand(edgeDeleteCmd)
}
