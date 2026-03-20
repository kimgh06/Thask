package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thask/cli/internal/output"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authentication commands",
}

var authWhoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show current authenticated user",
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := apiClient.Get("/api/auth/me")
		if err != nil {
			return err
		}
		if getFormat() == output.FormatJSON {
			output.JSON(data)
		} else {
			var user struct {
				ID          string `json:"id"`
				Email       string `json:"email"`
				DisplayName string `json:"displayName"`
			}
			json.Unmarshal(data, &user)
			fmt.Printf("%s (%s)\n", user.DisplayName, user.Email)
		}
		return nil
	},
}

func init() {
	authCmd.AddCommand(authWhoamiCmd)
}
