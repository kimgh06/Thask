package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thask/cli/internal/scan"
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan a project (Go or TypeScript/JavaScript) and import its dependency graph",
	RunE: func(cmd *cobra.Command, args []string) error {
		path, _ := cmd.Flags().GetString("path")
		maxFiles, _ := cmd.Flags().GetInt("max-files")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		plugin, _ := cmd.Flags().GetString("plugin")
		language, _ := cmd.Flags().GetString("language")

		var result *scan.ScanResult
		var err error
		if plugin != "" {
			result, err = scan.RunPlugin(plugin, path, maxFiles)
		} else {
			result, err = scan.Run(scan.ScanOptions{Path: path, MaxFiles: maxFiles, Language: language})
		}
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
	scanCmd.Flags().String("path", ".", "Path to project root (go.mod for Go, package.json for TS/JS)")
	scanCmd.Flags().Int("max-files", 500, "Maximum number of source files to scan")
	scanCmd.Flags().Bool("dry-run", false, "Print JSON to stdout instead of posting to API")
	scanCmd.Flags().String("plugin", "", "Path to external scanner plugin executable")
	scanCmd.Flags().String("language", "", "Force scanner language: '', 'go', or 'ts' (auto-detect when empty)")

	rootCmd.AddCommand(scanCmd)
}
