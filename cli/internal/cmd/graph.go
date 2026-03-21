package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/thask/cli/internal/output"
)

var graphCmd = &cobra.Command{
	Use:     "graph",
	Aliases: []string{"g"},
	Short:   "Graph operations",
}

var graphGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get full graph (nodes + edges)",
	RunE: func(cmd *cobra.Command, args []string) error {
		pid := resolveProject()
		if pid == "" {
			return fmt.Errorf("--project or THASK_PROJECT required")
		}
		data, err := apiClient.Get("/api/projects/" + pid + "/graph")
		if err != nil {
			return err
		}
		output.JSON(data)
		return nil
	},
}

var graphExportCmd = &cobra.Command{
	Use:     "export",
	Aliases: []string{"ex"},
	Short:   "Export graph to JSON file",
	RunE: func(cmd *cobra.Command, args []string) error {
		pid := resolveProject()
		if pid == "" {
			return fmt.Errorf("--project or THASK_PROJECT required")
		}
		data, err := apiClient.Get("/api/projects/" + pid + "/graph")
		if err != nil {
			return err
		}

		filePath, _ := cmd.Flags().GetString("file")
		if filePath == "" {
			filePath = "graph.json"
		}

		// Pretty-print JSON
		var raw any
		json.Unmarshal(data, &raw)
		out, _ := json.MarshalIndent(raw, "", "  ")

		if err := os.WriteFile(filePath, out, 0644); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
		fmt.Printf("Exported to %s\n", filePath)
		return nil
	},
}

var graphImportCmd = &cobra.Command{
	Use:   "import",
	Short: "Import graph from JSON file",
	RunE: func(cmd *cobra.Command, args []string) error {
		pid := resolveProject()
		if pid == "" {
			return fmt.Errorf("--project or THASK_PROJECT required")
		}
		filePath, _ := cmd.Flags().GetString("file")
		mode, _ := cmd.Flags().GetString("mode")

		fileData, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}

		var parsed struct {
			Nodes json.RawMessage `json:"nodes"`
			Edges json.RawMessage `json:"edges"`
		}
		if err := json.Unmarshal(fileData, &parsed); err != nil {
			return fmt.Errorf("invalid JSON: %w", err)
		}

		body := map[string]any{
			"mode":  mode,
			"nodes": json.RawMessage(parsed.Nodes),
			"edges": json.RawMessage(parsed.Edges),
		}

		data, err := apiClient.Post("/api/projects/"+pid+"/graph/import", body)
		if err != nil {
			return err
		}
		fmt.Println("Import successful")
		_ = data
		return nil
	},
}

var graphLayoutCmd = &cobra.Command{
	Use:     "layout",
	Aliases: []string{"l"},
	Short:   "Run auto-layout on the graph (repositions nodes, auto-sizes GROUPs)",
	RunE: func(cmd *cobra.Command, args []string) error {
		pid := resolveProject()
		if pid == "" {
			return fmt.Errorf("--project or THASK_PROJECT required")
		}
		algo, _ := cmd.Flags().GetString("algorithm")
		preserve, _ := cmd.Flags().GetBool("preserve-edges")
		body := map[string]any{}
		if algo != "" {
			body["algorithm"] = algo
		}
		if preserve {
			body["preserveEdges"] = true
		}
		data, err := apiClient.Post("/api/projects/"+pid+"/graph/layout", body)
		if err != nil {
			return err
		}
		switch getFormat() {
		case output.FormatQuiet:
			fmt.Println("Layout applied")
		default:
			output.JSON(data)
		}
		return nil
	},
}

func init() {
	graphImportCmd.Flags().String("file", "", "JSON file path (required)")
	graphImportCmd.Flags().String("mode", "merge", "Import mode: replace or merge")
	_ = graphImportCmd.MarkFlagRequired("file")

	graphExportCmd.Flags().String("file", "graph.json", "Output file path")

	graphLayoutCmd.Flags().String("algorithm", "", "Layout algorithm: dagre (default), grid")
	graphLayoutCmd.Flags().Bool("preserve-edges", false, "Preserve edge waypoints during layout")

	graphCmd.AddCommand(graphGetCmd)
	graphCmd.AddCommand(graphExportCmd)
	graphCmd.AddCommand(graphImportCmd)
	graphCmd.AddCommand(graphLayoutCmd)
}

// runLayoutIfFlagged calls layout endpoint if --layout flag is set
func runLayoutIfFlagged(cmd *cobra.Command) {
	layout, _ := cmd.Flags().GetBool("layout")
	if !layout {
		return
	}
	pid := resolveProject()
	if pid == "" {
		return
	}
	_, _ = apiClient.Post("/api/projects/"+pid+"/graph/layout", map[string]any{})
}
