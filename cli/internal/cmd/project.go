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

var projectShareCmd = &cobra.Command{
	Use:   "share",
	Short: "Enable link sharing (viewer by default)",
	RunE: func(cmd *cobra.Command, args []string) error {
		pid := resolveProject()
		if pid == "" {
			return fmt.Errorf("--project or THASK_PROJECT required")
		}
		mode, _ := cmd.Flags().GetString("mode")
		if mode == "" {
			mode = "viewer"
		}
		data, err := apiClient.Put("/api/projects/"+pid+"/sharing", map[string]string{"linkSharing": mode})
		if err != nil {
			return err
		}
		var project struct {
			ShareToken *string `json:"shareToken"`
		}
		json.Unmarshal(data, &project)
		if project.ShareToken != nil {
			fmt.Printf("Share link: /shared/%s\n", *project.ShareToken)
		}
		return nil
	},
}

var projectUnshareCmd = &cobra.Command{
	Use:   "unshare",
	Short: "Disable link sharing",
	RunE: func(cmd *cobra.Command, args []string) error {
		pid := resolveProject()
		if pid == "" {
			return fmt.Errorf("--project or THASK_PROJECT required")
		}
		_, err := apiClient.Put("/api/projects/"+pid+"/sharing", map[string]string{"linkSharing": "off"})
		if err != nil {
			return err
		}
		fmt.Println("Link sharing disabled")
		return nil
	},
}

var projectMembersCmd = &cobra.Command{
	Use:   "members",
	Short: "List project members",
	RunE: func(cmd *cobra.Command, args []string) error {
		pid := resolveProject()
		if pid == "" {
			return fmt.Errorf("--project or THASK_PROJECT required")
		}
		data, err := apiClient.Get("/api/projects/" + pid + "/sharing")
		if err != nil {
			return err
		}
		output.JSON(data)
		return nil
	},
}

var projectInviteCmd = &cobra.Command{
	Use:   "invite",
	Short: "Invite a user to the project",
	RunE: func(cmd *cobra.Command, args []string) error {
		pid := resolveProject()
		if pid == "" {
			return fmt.Errorf("--project or THASK_PROJECT required")
		}
		email, _ := cmd.Flags().GetString("email")
		role, _ := cmd.Flags().GetString("role")
		if role == "" {
			role = "viewer"
		}
		_, err := apiClient.Post("/api/projects/"+pid+"/sharing/members", map[string]string{
			"email": email, "role": role,
		})
		if err != nil {
			return err
		}
		fmt.Println("Invited successfully")
		return nil
	},
}

var projectKickCmd = &cobra.Command{
	Use:   "kick",
	Short: "Remove a user from the project",
	RunE: func(cmd *cobra.Command, args []string) error {
		pid := resolveProject()
		if pid == "" {
			return fmt.Errorf("--project or THASK_PROJECT required")
		}
		userID, _ := cmd.Flags().GetString("user")
		_, err := apiClient.Delete("/api/projects/" + pid + "/sharing/members/" + userID)
		if err != nil {
			return err
		}
		fmt.Println("Removed successfully")
		return nil
	},
}

func init() {
	projectCreateCmd.Flags().String("name", "", "Project name (required)")
	projectCreateCmd.Flags().String("description", "", "Project description")
	_ = projectCreateCmd.MarkFlagRequired("name")

	projectShareCmd.Flags().String("mode", "viewer", "Sharing mode: viewer or editor")

	projectInviteCmd.Flags().String("email", "", "Email to invite (required)")
	projectInviteCmd.Flags().String("role", "viewer", "Role: editor, viewer")
	_ = projectInviteCmd.MarkFlagRequired("email")

	projectKickCmd.Flags().String("user", "", "User ID to remove (required)")
	_ = projectKickCmd.MarkFlagRequired("user")

	projectCmd.AddCommand(projectListCmd)
	projectCmd.AddCommand(projectCreateCmd)
	projectCmd.AddCommand(projectGetCmd)
	projectCmd.AddCommand(projectShareCmd)
	projectCmd.AddCommand(projectUnshareCmd)
	projectCmd.AddCommand(projectMembersCmd)
	projectCmd.AddCommand(projectInviteCmd)
	projectCmd.AddCommand(projectKickCmd)
}
