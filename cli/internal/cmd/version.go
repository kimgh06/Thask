package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version and Commit are stamped at build time via -ldflags
// (see Makefile build-cli target). The `--version` flag and `version`
// subcommand both render the same output for first-time UX consistency.
var (
	Version = "dev"
	Commit  = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version info",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("thask %s (%s)\n", Version, Commit)
	},
}
