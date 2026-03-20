package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thask/cli/internal/output"
)

var teamCmd = &cobra.Command{
	Use:     "team",
	Aliases: []string{"t"},
	Short:   "Team management",
}

var teamListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all teams",
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := apiClient.Get("/api/teams")
		if err != nil {
			return err
		}
		switch getFormat() {
		case output.FormatTable:
			var teams []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
				Slug string `json:"slug"`
			}
			json.Unmarshal(data, &teams)
			rows := make([][]string, len(teams))
			for i, t := range teams {
				rows[i] = []string{t.ID[:8], t.Slug, t.Name}
			}
			output.Table([]string{"ID", "SLUG", "NAME"}, rows)
		case output.FormatQuiet:
			var teams []struct{ ID string `json:"id"` }
			json.Unmarshal(data, &teams)
			ids := make([]string, len(teams))
			for i, t := range teams {
				ids[i] = t.ID
			}
			output.Quiet(ids)
		default:
			output.JSON(data)
		}
		return nil
	},
}

var teamGetCmd = &cobra.Command{
	Use:   "get <slug>",
	Short: "Get team details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := apiClient.Get("/api/teams/" + args[0])
		if err != nil {
			return err
		}
		output.JSON(data)
		return nil
	},
}

var teamCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new team",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		slug, _ := cmd.Flags().GetString("slug")
		data, err := apiClient.Post("/api/teams", map[string]string{"name": name, "slug": slug})
		if err != nil {
			return err
		}
		output.JSON(data)
		return nil
	},
}

var teamMembersCmd = &cobra.Command{
	Use:   "members <slug>",
	Short: "List team members",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := apiClient.Get("/api/teams/" + args[0] + "/members")
		if err != nil {
			return err
		}
		switch getFormat() {
		case output.FormatTable:
			var members []struct {
				Role string `json:"role"`
				User struct {
					ID          string `json:"id"`
					Email       string `json:"email"`
					DisplayName string `json:"displayName"`
				} `json:"user"`
			}
			json.Unmarshal(data, &members)
			rows := make([][]string, len(members))
			for i, m := range members {
				rows[i] = []string{m.User.ID[:8], m.User.DisplayName, m.User.Email, m.Role}
			}
			output.Table([]string{"ID", "NAME", "EMAIL", "ROLE"}, rows)
		default:
			output.JSON(data)
		}
		return nil
	},
}

var teamInviteCmd = &cobra.Command{
	Use:   "invite <slug>",
	Short: "Invite a member to team",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		email, _ := cmd.Flags().GetString("email")
		role, _ := cmd.Flags().GetString("role")
		body := map[string]string{"email": email}
		if role != "" {
			body["role"] = role
		}
		data, err := apiClient.Post("/api/teams/"+args[0]+"/members", body)
		if err != nil {
			return err
		}
		fmt.Println("Invited successfully")
		_ = data
		return nil
	},
}

func init() {
	teamCreateCmd.Flags().String("name", "", "Team name (required)")
	teamCreateCmd.Flags().String("slug", "", "Team slug (required)")
	_ = teamCreateCmd.MarkFlagRequired("name")
	_ = teamCreateCmd.MarkFlagRequired("slug")

	teamInviteCmd.Flags().String("email", "", "Email to invite (required)")
	teamInviteCmd.Flags().String("role", "", "Role: admin, member, viewer")
	_ = teamInviteCmd.MarkFlagRequired("email")

	teamCmd.AddCommand(teamListCmd)
	teamCmd.AddCommand(teamGetCmd)
	teamCmd.AddCommand(teamCreateCmd)
	teamCmd.AddCommand(teamMembersCmd)
	teamCmd.AddCommand(teamInviteCmd)
}
