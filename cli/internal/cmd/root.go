package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/thask/cli/internal/client"
	"github.com/thask/cli/internal/config"
	"github.com/thask/cli/internal/output"
)

var (
	cfg       *config.Config
	apiClient *client.Client
	fmtFlag   string
	prettyFlag bool
	quietFlag  bool
	urlFlag    string
	tokenFlag  string
	projectFlag string
	teamFlag   string
)

var rootCmd = &cobra.Command{
	Use:   "thask",
	Short: "Thask CLI — graph-based task management for humans and AI agents",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		cfg = config.Load()

		// Flag overrides
		if urlFlag != "" {
			cfg.URL = urlFlag
		}
		if tokenFlag != "" {
			cfg.Token = tokenFlag
		}
		if projectFlag != "" {
			cfg.Project = projectFlag
		}
		if teamFlag != "" {
			cfg.Team = teamFlag
		}

		// Commands that don't need auth
		// Commands that don't need auth
		skipAuth := map[string]bool{
			"config": true, "set": true, "show": true, "version": true,
			"serve": true, "aliases": true, "install": true, "uninstall": true,
			"guide": true,
		}
		if skipAuth[cmd.Name()] {
			return nil
		}

		if err := cfg.Validate(); err != nil {
			return err
		}

		apiClient = client.New(cfg.URL, cfg.Token)
		return nil
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&urlFlag, "url", "", "Backend URL (env: THASK_URL)")
	rootCmd.PersistentFlags().StringVar(&tokenFlag, "token", "", "API token (env: THASK_TOKEN)")
	rootCmd.PersistentFlags().StringVarP(&projectFlag, "project", "p", "", "Project ID (env: THASK_PROJECT)")
	rootCmd.PersistentFlags().StringVar(&teamFlag, "team", "", "Team slug (env: THASK_TEAM)")
	rootCmd.PersistentFlags().StringVarP(&fmtFlag, "format", "f", "json", "Output format: json, table, quiet")
	rootCmd.PersistentFlags().BoolVar(&prettyFlag, "pretty", false, "Shorthand for --format table")
	rootCmd.PersistentFlags().BoolVarP(&quietFlag, "quiet", "q", false, "Shorthand for --format quiet")

	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(teamCmd)
	rootCmd.AddCommand(projectCmd)
	rootCmd.AddCommand(nodeCmd)
	rootCmd.AddCommand(edgeCmd)
	rootCmd.AddCommand(graphCmd)
	rootCmd.AddCommand(impactCmd)
	rootCmd.AddCommand(mcpCmd)
	rootCmd.AddCommand(aliasesCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(guideCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		output.ErrorJSON(err.Error())
		os.Exit(client.ExitCode(err))
	}
}

func getFormat() output.Format {
	return output.ParseFormat(fmtFlag, prettyFlag, quietFlag)
}

func resolveProject() string {
	if projectFlag != "" {
		return projectFlag
	}
	return cfg.Project
}

func resolveTeam() string {
	if teamFlag != "" {
		return teamFlag
	}
	return cfg.Team
}
