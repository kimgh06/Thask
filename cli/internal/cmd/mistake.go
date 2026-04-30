package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thask/cli/internal/mcp"
	"github.com/thask/cli/internal/output"
)

var mistakeCmd = &cobra.Command{
	Use:     "mistake",
	Aliases: []string{"mistakes"},
	Short:   "Record agent mistakes for self-reinforcing learning",
	Long:    "Record a mistake under the project's '실수 기록' GROUP. Future thask.guide calls will surface these so the same error is less likely to repeat.",
}

var mistakeRecordCmd = &cobra.Command{
	Use:   "record",
	Short: "Record a single mistake as a BUG node",
	Long: `Records a mistake as a BUG node (status=FAIL) under the project's '실수 기록' GROUP, auto-creating the group if missing.

Example:
  thask mistake record \
    --title "Used invalid status TODO" \
    --cause "Did not check valid enum before creating node" \
    --fix "Removed --status flag, defaults to IN_PROGRESS" \
    --lesson "Confirm enum values before using new flags"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		pid := resolveProject()
		if pid == "" {
			return fmt.Errorf("--project or THASK_PROJECT required")
		}
		title, _ := cmd.Flags().GetString("title")
		lesson, _ := cmd.Flags().GetString("lesson")
		if title == "" || lesson == "" {
			return fmt.Errorf("--title and --lesson are required")
		}
		cause, _ := cmd.Flags().GetString("cause")
		fix, _ := cmd.Flags().GetString("fix")

		result, err := mcp.HandleToolCall(apiClient, "thask.mistake.record", mustJSON(map[string]any{
			"projectId": pid,
			"title":     title,
			"cause":     cause,
			"fix":       fix,
			"lesson":    lesson,
		}))
		if err != nil {
			return err
		}

		switch getFormat() {
		case output.FormatQuiet:
			if m, ok := result.(map[string]any); ok {
				if id, _ := m["id"].(string); id != "" {
					fmt.Println(id)
					return nil
				}
			}
		case output.FormatTable:
			if m, ok := result.(map[string]any); ok {
				output.Table([]string{"ID", "TITLE", "GROUP"}, [][]string{{
					stringOr(m, "id"), stringOr(m, "title"), stringOr(m, "groupId"),
				}})
				return nil
			}
		}
		raw, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(raw))
		return nil
	},
}

func mustJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

func stringOr(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func init() {
	mistakeRecordCmd.Flags().String("title", "", "Short label for the mistake (required)")
	mistakeRecordCmd.Flags().String("cause", "", "Why it happened — wrong assumption or missing check")
	mistakeRecordCmd.Flags().String("fix", "", "How it was corrected this time")
	mistakeRecordCmd.Flags().String("lesson", "", "Rule for future sessions to avoid repeating it (required)")
	mistakeCmd.AddCommand(mistakeRecordCmd)
}
