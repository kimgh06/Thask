package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thask/cli/internal/output"
)

var suggestionsCmd = &cobra.Command{
	Use:     "suggestions",
	Aliases: []string{"sug"},
	Short:   "Review and decide agent-proposed node changes",
}

var suggestionsListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List pending (and optionally decided) suggestions for the project",
	RunE: func(cmd *cobra.Command, args []string) error {
		pid := resolveProject()
		if pid == "" {
			return fmt.Errorf("--project or THASK_PROJECT required")
		}
		path := "/api/projects/" + pid + "/suggestions"
		status, _ := cmd.Flags().GetString("status")
		limit, _ := cmd.Flags().GetInt("limit")
		query := ""
		if status != "" {
			query = "status=" + status
		}
		if limit > 0 {
			if query != "" {
				query += "&"
			}
			query += fmt.Sprintf("limit=%d", limit)
		}
		if query != "" {
			path += "?" + query
		}
		data, err := apiClient.Get(path)
		if err != nil {
			return err
		}
		output.JSON(data)
		return nil
	},
}

var suggestionsDecideCmd = &cobra.Command{
	Use:   "decide <suggestionId>",
	Short: "Accept or reject a pending suggestion (human only)",
	Long: `Decides one suggestion. Pass --accept or --reject (mutually exclusive).
The server enforces actor_kind=user_interactive, so this command works only
with a human session or a non-agent API key.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pid := resolveProject()
		if pid == "" {
			return fmt.Errorf("--project or THASK_PROJECT required")
		}
		accept, _ := cmd.Flags().GetBool("accept")
		reject, _ := cmd.Flags().GetBool("reject")
		if accept == reject {
			return fmt.Errorf("supply exactly one of --accept or --reject")
		}
		status := "accepted"
		if reject {
			status = "rejected"
		}
		body := map[string]any{"status": status}
		if reason, _ := cmd.Flags().GetString("reason"); reason != "" {
			body["reason"] = reason
		}
		data, err := apiClient.Patch("/api/projects/"+pid+"/suggestions/"+args[0], body)
		if err != nil {
			return err
		}
		output.JSON(data)
		return nil
	},
}

func init() {
	suggestionsListCmd.Flags().String("status", "", "Filter by status: pending|accepted|rejected|expired|superseded")
	suggestionsListCmd.Flags().Int("limit", 0, "Max rows (default: server choice)")

	suggestionsDecideCmd.Flags().Bool("accept", false, "Accept the suggestion and apply the proposed change")
	suggestionsDecideCmd.Flags().Bool("reject", false, "Reject the suggestion (no change applied)")
	suggestionsDecideCmd.Flags().String("reason", "", "Optional decision reason recorded on the suggestion row")

	suggestionsCmd.AddCommand(suggestionsListCmd)
	suggestionsCmd.AddCommand(suggestionsDecideCmd)
}
