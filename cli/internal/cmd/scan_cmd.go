package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thask/cli/internal/scan"
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan a Go project and import its dependency graph",
	RunE: func(cmd *cobra.Command, args []string) error {
		path, _ := cmd.Flags().GetString("path")
		maxFiles, _ := cmd.Flags().GetInt("max-files")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		result, err := scan.Run(scan.ScanOptions{
			Path:     path,
			MaxFiles: maxFiles,
		})
		if err != nil {
			return err
		}

		if dryRun {
			out, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(out))
			return nil
		}

		pid := resolveProject()
		if pid == "" {
			return fmt.Errorf("--project or THASK_PROJECT required (or use --dry-run)")
		}

		_, err = apiClient.Post("/api/projects/"+pid+"/graph/import", result)
		if err != nil {
			return err
		}
		fmt.Printf("Imported %d nodes and %d edges\n", len(result.Nodes), len(result.Edges))
		return nil
	},
}

func init() {
	scanCmd.Flags().String("path", ".", "Path to Go project root (must contain go.mod)")
	scanCmd.Flags().Int("max-files", 500, "Maximum number of .go files to scan")
	scanCmd.Flags().Bool("dry-run", false, "Print JSON to stdout instead of posting to API")

	rootCmd.AddCommand(scanCmd)
}
