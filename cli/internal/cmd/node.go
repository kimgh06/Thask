package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/thask/cli/internal/output"
)

var nodeCmd = &cobra.Command{
	Use:     "node",
	Aliases: []string{"n"},
	Short:   "Node operations",
}

var nodeListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List nodes in a project",
	RunE: func(cmd *cobra.Command, args []string) error {
		pid := resolveProject()
		if pid == "" {
			return fmt.Errorf("--project or THASK_PROJECT required")
		}
		path := "/api/projects/" + pid + "/nodes"
		var params []string
		if t, _ := cmd.Flags().GetString("type"); t != "" {
			params = append(params, "type="+t)
		}
		if s, _ := cmd.Flags().GetString("status"); s != "" {
			params = append(params, "status="+s)
		}
		if len(params) > 0 {
			path += "?" + strings.Join(params, "&")
		}
		data, err := apiClient.Get(path)
		if err != nil {
			return err
		}
		switch getFormat() {
		case output.FormatTable:
			var nodes []struct {
				ID     string `json:"id"`
				Type   string `json:"type"`
				Title  string `json:"title"`
				Status string `json:"status"`
			}
			json.Unmarshal(data, &nodes)
			rows := make([][]string, len(nodes))
			for i, n := range nodes {
				title := n.Title
				if len(title) > 40 {
					title = title[:37] + "..."
				}
				rows[i] = []string{n.ID[:8], n.Type, title, n.Status}
			}
			output.Table([]string{"ID", "TYPE", "TITLE", "STATUS"}, rows)
		case output.FormatQuiet:
			var nodes []struct{ ID string `json:"id"` }
			json.Unmarshal(data, &nodes)
			ids := make([]string, len(nodes))
			for i, n := range nodes {
				ids[i] = n.ID
			}
			output.Quiet(ids)
		default:
			output.JSON(data)
		}
		return nil
	},
}

var nodeGetCmd = &cobra.Command{
	Use:     "get <nodeId>",
	Aliases: []string{"g"},
	Short:   "Get node details with history",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pid := resolveProject()
		if pid == "" {
			return fmt.Errorf("--project or THASK_PROJECT required")
		}
		data, err := apiClient.Get("/api/projects/" + pid + "/nodes/" + args[0])
		if err != nil {
			return err
		}
		output.JSON(data)
		return nil
	},
}

var nodeCreateCmd = &cobra.Command{
	Use:     "create",
	Aliases: []string{"c"},
	Short:   "Create a node",
	RunE: func(cmd *cobra.Command, args []string) error {
		pid := resolveProject()
		if pid == "" {
			return fmt.Errorf("--project or THASK_PROJECT required")
		}
		nodeType, _ := cmd.Flags().GetString("type")
		title, _ := cmd.Flags().GetString("title")
		desc, _ := cmd.Flags().GetString("description")
		status, _ := cmd.Flags().GetString("status")
		tagsStr, _ := cmd.Flags().GetString("tags")
		x, _ := cmd.Flags().GetFloat64("x")
		y, _ := cmd.Flags().GetFloat64("y")

		body := map[string]any{
			"type":      nodeType,
			"title":     title,
			"positionX": x,
			"positionY": y,
		}
		if desc != "" {
			body["description"] = desc
		}
		if status != "" {
			body["status"] = status
		}
		if tagsStr != "" {
			body["tags"] = strings.Split(tagsStr, ",")
		}

		data, err := apiClient.Post("/api/projects/"+pid+"/nodes", body)
		if err != nil {
			return err
		}
		switch getFormat() {
		case output.FormatQuiet:
			var node struct{ ID string `json:"id"` }
			json.Unmarshal(data, &node)
			fmt.Println(node.ID)
		default:
			output.JSON(data)
		}
		runLayoutIfFlagged(cmd)
		return nil
	},
}

var nodeUpdateCmd = &cobra.Command{
	Use:     "update <nodeId>",
	Aliases: []string{"u"},
	Short:   "Update a node",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pid := resolveProject()
		if pid == "" {
			return fmt.Errorf("--project or THASK_PROJECT required")
		}
		body := map[string]any{}
		if cmd.Flags().Changed("title") {
			v, _ := cmd.Flags().GetString("title")
			body["title"] = v
		}
		if cmd.Flags().Changed("status") {
			v, _ := cmd.Flags().GetString("status")
			body["status"] = v
		}
		if cmd.Flags().Changed("type") {
			v, _ := cmd.Flags().GetString("type")
			body["type"] = v
		}
		if cmd.Flags().Changed("description") {
			v, _ := cmd.Flags().GetString("description")
			body["description"] = v
		}
		if cmd.Flags().Changed("tags") {
			v, _ := cmd.Flags().GetString("tags")
			body["tags"] = strings.Split(v, ",")
		}
		if cmd.Flags().Changed("parent") {
			v, _ := cmd.Flags().GetString("parent")
			if v == "" || v == "null" || v == "none" {
				body["parentId"] = nil
			} else {
				body["parentId"] = v
			}
		}
		if len(body) == 0 {
			return fmt.Errorf("no fields to update")
		}
		data, err := apiClient.Patch("/api/projects/"+pid+"/nodes/"+args[0], body)
		if err != nil {
			return err
		}
		output.JSON(data)
		return nil
	},
}

var nodeDeleteCmd = &cobra.Command{
	Use:     "delete <nodeId>",
	Aliases: []string{"d", "rm"},
	Short:   "Delete a node",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pid := resolveProject()
		if pid == "" {
			return fmt.Errorf("--project or THASK_PROJECT required")
		}
		_, err := apiClient.Delete("/api/projects/" + pid + "/nodes/" + args[0])
		if err != nil {
			return err
		}
		fmt.Println("Deleted")
		runLayoutIfFlagged(cmd)
		return nil
	},
}

var nodeBatchStatusCmd = &cobra.Command{
	Use:   "batch-status",
	Short: "Batch update node status",
	RunE: func(cmd *cobra.Command, args []string) error {
		pid := resolveProject()
		if pid == "" {
			return fmt.Errorf("--project or THASK_PROJECT required")
		}
		idsStr, _ := cmd.Flags().GetString("ids")
		status, _ := cmd.Flags().GetString("status")
		ids := strings.Split(idsStr, ",")
		data, err := apiClient.Patch("/api/projects/"+pid+"/nodes/batch-status", map[string]any{
			"ids": ids, "status": status,
		})
		if err != nil {
			return err
		}
		output.JSON(data)
		return nil
	},
}

func init() {
	nodeListCmd.Flags().String("type", "", "Filter by type: TASK, BUG, FLOW, etc.")
	nodeListCmd.Flags().String("status", "", "Filter by status: PASS, FAIL, IN_PROGRESS, BLOCKED")

	nodeCreateCmd.Flags().String("type", "", "Node type (required)")
	nodeCreateCmd.Flags().String("title", "", "Node title (required)")
	nodeCreateCmd.Flags().String("description", "", "Description")
	nodeCreateCmd.Flags().String("status", "", "Status")
	nodeCreateCmd.Flags().String("tags", "", "Comma-separated tags")
	nodeCreateCmd.Flags().Float64("x", 0, "Position X")
	nodeCreateCmd.Flags().Float64("y", 0, "Position Y")
	nodeCreateCmd.Flags().Bool("layout", false, "Run auto-layout after creation")
	_ = nodeCreateCmd.MarkFlagRequired("type")
	_ = nodeCreateCmd.MarkFlagRequired("title")

	nodeUpdateCmd.Flags().String("title", "", "New title")
	nodeUpdateCmd.Flags().String("status", "", "New status")
	nodeUpdateCmd.Flags().String("type", "", "New type")
	nodeUpdateCmd.Flags().String("description", "", "New description")
	nodeUpdateCmd.Flags().String("tags", "", "Comma-separated tags")
	nodeUpdateCmd.Flags().String("parent", "", "Parent node ID (GROUP), 'none' to unparent")

	nodeDeleteCmd.Flags().Bool("layout", false, "Run auto-layout after deletion")

	nodeBatchStatusCmd.Flags().String("ids", "", "Comma-separated node IDs (required)")
	nodeBatchStatusCmd.Flags().String("status", "", "New status (required)")
	_ = nodeBatchStatusCmd.MarkFlagRequired("ids")
	_ = nodeBatchStatusCmd.MarkFlagRequired("status")

	nodeCmd.AddCommand(nodeListCmd)
	nodeCmd.AddCommand(nodeGetCmd)
	nodeCmd.AddCommand(nodeCreateCmd)
	nodeCmd.AddCommand(nodeUpdateCmd)
	nodeCmd.AddCommand(nodeDeleteCmd)
	nodeCmd.AddCommand(nodeBatchStatusCmd)
}
