package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/thask/cli/internal/client"
	"github.com/thask/cli/internal/config"
	"github.com/thask/cli/internal/output"
	"github.com/thask/cli/internal/update"
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
		// Skip update check for MCP serve (long-running stdio process —
		// stray stderr output may confuse the connected agent).
		if cmd.CommandPath() != "thask mcp serve" {
			cleanup := update.Check(Version)
			cobra.OnFinalize(cleanup)
		}

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
			"guide": true, "init": true, "doctor": true, "login": true,
		}
		if skipAuth[cmd.Name()] {
			return nil
		}

		if err := cfg.Validate(); err != nil {
			return err
		}

		apiClient = client.New(cfg.URL, cfg.Token)
		apiClient.ClientHeader = "thask-cli/" + Version
		return nil
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	// Wire native --version / -v handling so first-time users get a
	// readable response instead of {"error":"unknown flag: --version"}.
	rootCmd.Version = Version + " (" + Commit + ")"
	rootCmd.SetVersionTemplate("thask {{.Version}}\n")

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
	rootCmd.AddCommand(mistakeCmd)
}

// usageErrorPrefixes match Cobra's flag-parser / unknown-command / arg-count
// failures. These are user typos, not API/data failures — humans get a
// readable stderr message instead of a JSON envelope. Anything not matched
// falls through to ErrorJSON so the MCP/script contract is preserved.
var usageErrorPrefixes = []string{
	"unknown command",
	"unknown flag",
	"unknown shorthand flag",
	"flag provided but not defined",
	"flag needs an argument",
	"required flag(s)",
	"invalid argument",
	"bad flag syntax",
}

func isUsageError(err error) bool {
	msg := err.Error()
	for _, p := range usageErrorPrefixes {
		if strings.HasPrefix(msg, p) {
			return true
		}
	}
	// Argcount enforcement errors:
	//   "accepts at most N arg(s), received M" (cobra.MaximumNArgs)
	//   "accepts %d arg(s), received %d"       (cobra.ExactArgs / RangeArgs)
	//   "requires at least N arg(s)"           (cobra.MinimumNArgs)
	//   "accepts no arg(s)"                    (cobra.NoArgs)
	//   "requires exactly"                     (variants)
	// `arg(s), received` is the most reliable common substring across
	// all of Cobra's ArgFunc messages — use it as a catch-all.
	if strings.Contains(msg, "arg(s), received") ||
		strings.Contains(msg, "accepts at most") || strings.Contains(msg, "requires at least") ||
		strings.Contains(msg, "accepts no") || strings.Contains(msg, "requires exactly") {
		return true
	}
	// pflag parse failures bubble up wrapped with strconv. prefix e.g.
	// `strconv.ParseBool: parsing "yep": invalid syntax`. Treat as usage.
	if strings.Contains(msg, "strconv.") {
		return true
	}
	return false
}

func Execute() {
	err := rootCmd.Execute()
	if err == nil {
		return
	}
	if isUsageError(err) {
		fmt.Fprintf(os.Stderr, "Error: %s\nRun 'thask --help' for usage.\n", err)
		os.Exit(2)
	}
	output.ErrorJSON(err.Error())
	os.Exit(client.ExitCode(err))
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
