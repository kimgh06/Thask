package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

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

var graphCaptureCmd = &cobra.Command{
	Use:     "capture",
	Aliases: []string{"cap"},
	Short:   "Capture the graph as an image",
	RunE: func(cmd *cobra.Command, args []string) error {
		pid := resolveProject()
		if pid == "" {
			return fmt.Errorf("--project or THASK_PROJECT required")
		}

		filePath, _ := cmd.Flags().GetString("out")
		if filePath == "" {
			filePath, _ = cmd.Flags().GetString("file")
		}
		format, _ := cmd.Flags().GetString("format")
		width, _ := cmd.Flags().GetInt("width")
		height, _ := cmd.Flags().GetInt("height")
		padding, _ := cmd.Flags().GetInt("padding")
		scale, _ := cmd.Flags().GetInt("scale")
		crop, _ := cmd.Flags().GetBool("crop")

		if format == "" {
			format = "png"
		}
		if format != "png" && format != "svg" {
			return fmt.Errorf("only --format png or --format svg is supported")
		}
		if filePath == "" {
			if format == "svg" {
				filePath = "graph.svg"
			} else {
				filePath = "graph.png"
			}
		}

		q := url.Values{}
		q.Set("format", format)
		if width > 0 {
			q.Set("width", fmt.Sprintf("%d", width))
		}
		if height > 0 {
			q.Set("height", fmt.Sprintf("%d", height))
		}
		if padding >= 0 {
			q.Set("padding", fmt.Sprintf("%d", padding))
		}
		if scale > 0 {
			q.Set("scale", fmt.Sprintf("%d", scale))
		}
		q.Set("crop", fmt.Sprintf("%t", crop))

		accept := "image/png"
		if format == "svg" {
			accept = "image/svg+xml"
		}
		data, _, err := apiClient.GetRaw("/api/v1/projects/"+pid+"/graph/capture?"+q.Encode(), accept)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filePath, data, 0644); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
		fmt.Printf("Captured to %s\n", filePath)
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

var graphAnalyzeCmd = &cobra.Command{
	Use:     "analyze",
	Aliases: []string{"a"},
	Short:   "Detect cycles and find critical path",
	RunE: func(cmd *cobra.Command, args []string) error {
		pid := resolveProject()
		if pid == "" {
			return fmt.Errorf("--project or THASK_PROJECT required")
		}
		format, _ := cmd.Flags().GetString("format")

		resp, err := apiClient.Get(fmt.Sprintf("/api/projects/%s/graph/analyze", pid))
		if err != nil {
			return err
		}

		if format == "json" {
			fmt.Println(string(resp))
			return nil
		}

		var result struct {
			Data struct {
				Cycles       [][]string `json:"cycles"`
				CriticalPath []string   `json:"criticalPath"`
			} `json:"data"`
		}
		if err := json.Unmarshal(resp, &result); err != nil {
			return err
		}

		fmt.Printf("Cycles: %d\n", len(result.Data.Cycles))
		for i, cycle := range result.Data.Cycles {
			fmt.Printf("  %d. %s\n", i+1, strings.Join(cycle, " -> "))
		}
		fmt.Println()
		fmt.Printf("Critical Path: %d nodes\n", len(result.Data.CriticalPath))
		if len(result.Data.CriticalPath) > 0 {
			fmt.Printf("  %s\n", strings.Join(result.Data.CriticalPath, " -> "))
		}

		return nil
	},
}

func init() {
	graphImportCmd.Flags().String("file", "", "JSON file path (required)")
	graphImportCmd.Flags().String("mode", "merge", "Import mode: replace or merge")
	_ = graphImportCmd.MarkFlagRequired("file")

	graphExportCmd.Flags().String("file", "graph.json", "Output file path")

	graphCaptureCmd.Flags().String("out", "", "Output image path")
	graphCaptureCmd.Flags().String("file", "", "Output image path (deprecated; use --out)")
	graphCaptureCmd.Flags().String("format", "png", "Image format: png or svg")
	graphCaptureCmd.Flags().Int("width", 1600, "Image width in pixels")
	graphCaptureCmd.Flags().Int("height", 1000, "Image height in pixels")
	graphCaptureCmd.Flags().Int("padding", 80, "Fit padding in pixels")
	graphCaptureCmd.Flags().Int("scale", 2, "Output pixel scale: 1-4")
	graphCaptureCmd.Flags().Bool("crop", true, "Crop PNG output to graph bounds")

	graphLayoutCmd.Flags().String("algorithm", "", "Layout algorithm: dagre (default), grid")
	graphLayoutCmd.Flags().Bool("preserve-edges", false, "Preserve edge waypoints during layout")

	graphAnalyzeCmd.Flags().String("format", "", "Output format: json (default: human-readable)")

	graphCmd.AddCommand(graphGetCmd)
	graphCmd.AddCommand(graphExportCmd)
	graphCmd.AddCommand(graphCaptureCmd)
	graphCmd.AddCommand(graphImportCmd)
	graphCmd.AddCommand(graphLayoutCmd)
	graphCmd.AddCommand(graphAnalyzeCmd)
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
