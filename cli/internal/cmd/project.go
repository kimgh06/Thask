package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thask/cli/internal/output"
)

var projectCmd = &cobra.Command{
	Use:     "project",
	Aliases: []string{"p"},
	Short:   "Project management",
}

var projectListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List projects in a team",
	RunE: func(cmd *cobra.Command, args []string) error {
		team := resolveTeam()
		if team == "" {
			return fmt.Errorf("--team or THASK_TEAM required")
		}
		data, err := apiClient.Get("/api/teams/" + team + "/projects")
		if err != nil {
			return err
		}
		switch getFormat() {
		case output.FormatTable:
			var projects []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			}
			json.Unmarshal(data, &projects)
			rows := make([][]string, len(projects))
			for i, p := range projects {
				rows[i] = []string{p.ID[:8], p.Name}
			}
			output.Table([]string{"ID", "NAME"}, rows)
		case output.FormatQuiet:
			var projects []struct{ ID string `json:"id"` }
			json.Unmarshal(data, &projects)
			ids := make([]string, len(projects))
			for i, p := range projects {
				ids[i] = p.ID
			}
			output.Quiet(ids)
		default:
			output.JSON(data)
		}
		return nil
	},
}

var projectCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a project",
	RunE: func(cmd *cobra.Command, args []string) error {
		team := resolveTeam()
		if team == "" {
			return fmt.Errorf("--team or THASK_TEAM required")
		}
		name, _ := cmd.Flags().GetString("name")
		desc, _ := cmd.Flags().GetString("description")
		body := map[string]any{"name": name}
		if desc != "" {
			body["description"] = desc
		}
		data, err := apiClient.Post("/api/teams/"+team+"/projects", body)
		if err != nil {
			return err
		}
		output.JSON(data)
		return nil
	},
}

var projectGetCmd = &cobra.Command{
	Use:   "get [projectId]",
	Short: "Get project details",
	RunE: func(cmd *cobra.Command, args []string) error {
		pid := resolveProject()
		if len(args) > 0 {
			pid = args[0]
		}
		if pid == "" {
			return fmt.Errorf("project ID required (arg or --project)")
		}
		data, err := apiClient.Get("/api/projects/" + pid)
		if err != nil {
			return err
		}
		output.JSON(data)
		return nil
	},
}

func init() {
	projectCreateCmd.Flags().String("name", "", "Project name (required)")
	projectCreateCmd.Flags().String("description", "", "Project description")
	_ = projectCreateCmd.MarkFlagRequired("name")

	projectCmd.AddCommand(projectListCmd)
	projectCmd.AddCommand(projectCreateCmd)
	projectCmd.AddCommand(projectGetCmd)
}
