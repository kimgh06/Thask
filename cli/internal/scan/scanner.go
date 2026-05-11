package scan

import (
	"context"
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type ScanNode struct {
	Type      string  `json:"type"`
	Title     string  `json:"title"`
	Status    string  `json:"status"`
	PositionX float64 `json:"positionX"`
	PositionY float64 `json:"positionY"`
}

type ScanEdge struct {
	SourceTitle string `json:"sourceTitle"`
	TargetTitle string `json:"targetTitle"`
	EdgeType    string `json:"edgeType"`
}

type ScanResult struct {
	Mode  string     `json:"mode"`
	Nodes []ScanNode `json:"nodes"`
	Edges []ScanEdge `json:"edges"`
}

type ScanOptions struct {
	Path     string
	MaxFiles int
	// Language selects the scanner. "" = auto-detect, "go", "ts"/"js"/"tsx"/"typescript"/"javascript".
	Language string
}

// LanguageScanner is implemented by per-language dependency scanners.
type LanguageScanner interface {
	Name() string
	Scan(opts ScanOptions) (*ScanResult, error)
}

// goScanner scans Go projects for inter-package dependencies via go.mod + AST imports.
type goScanner struct{}

func (goScanner) Name() string { return "go" }

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// detectScanner picks a language scanner based on opts.Language or by sniffing the project root.
func detectScanner(opts ScanOptions) LanguageScanner {
	switch strings.ToLower(opts.Language) {
	case "go":
		return &goScanner{}
	case "ts", "js", "tsx", "jsx", "javascript", "typescript":
		return &tsScanner{}
	}
	if fileExists(filepath.Join(opts.Path, "go.mod")) {
		return &goScanner{}
	}
	if fileExists(filepath.Join(opts.Path, "package.json")) {
		return &tsScanner{}
	}
	return &goScanner{}
}

// Run dispatches to the appropriate language scanner.
func Run(opts ScanOptions) (*ScanResult, error) {
	return detectScanner(opts).Scan(opts)
}

func (goScanner) Scan(opts ScanOptions) (*ScanResult, error) {
	if opts.MaxFiles <= 0 {
		opts.MaxFiles = 500
	}

	// 1. Find and parse go.mod
	goModPath := filepath.Join(opts.Path, "go.mod")
	modData, err := os.ReadFile(goModPath)
	if err != nil {
		return nil, fmt.Errorf("no go.mod found in %s", opts.Path)
	}
	modulePath := parseModulePath(string(modData))
	if modulePath == "" {
		return nil, fmt.Errorf("could not parse module path from go.mod")
	}

	// 2. Walk .go files and collect packages + imports
	fset := token.NewFileSet()
	pkgImports := make(map[string]map[string]bool) // pkg -> set of imported pkgs
	fileCount := 0

	err = filepath.WalkDir(opts.Path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		// Skip symlinks to prevent directory traversal
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		// Skip vendor, testdata, hidden dirs, _ prefixed dirs
		if d.IsDir() {
			base := d.Name()
			if base == "vendor" || base == "testdata" || strings.HasPrefix(base, ".") || strings.HasPrefix(base, "_") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".go") || strings.HasSuffix(d.Name(), "_test.go") {
			return nil
		}

		fileCount++
		if fileCount > opts.MaxFiles {
			return fmt.Errorf("exceeded maxFiles limit (%d/%d files found)", fileCount, opts.MaxFiles)
		}

		f, parseErr := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if parseErr != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping %s: %v\n", path, parseErr)
			return nil
		}

		// Determine this file's package path relative to module
		dir := filepath.Dir(path)
		relDir, _ := filepath.Rel(opts.Path, dir)
		if relDir == "." {
			relDir = ""
		}
		var pkgPath string
		if relDir == "" {
			pkgPath = modulePath
		} else {
			pkgPath = modulePath + "/" + filepath.ToSlash(relDir)
		}

		if pkgImports[pkgPath] == nil {
			pkgImports[pkgPath] = make(map[string]bool)
		}

		for _, imp := range f.Imports {
			impPath := strings.Trim(imp.Path.Value, `"`)
			// Only include intra-module imports
			if strings.HasPrefix(impPath, modulePath) && impPath != pkgPath {
				pkgImports[pkgPath][impPath] = true
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// 3. Build result
	// Create nodes for all packages
	pkgSet := make(map[string]bool)
	for pkg := range pkgImports {
		pkgSet[pkg] = true
		for dep := range pkgImports[pkg] {
			pkgSet[dep] = true
		}
	}

	nodes := make([]ScanNode, 0, len(pkgSet))
	i := 0
	for pkg := range pkgSet {
		title := pkgToTitle(pkg, modulePath)
		nodes = append(nodes, ScanNode{
			Type:      "TASK",
			Title:     title,
			Status:    "IN_PROGRESS",
			PositionX: float64(i%5) * 200,
			PositionY: float64(i/5) * 150,
		})
		i++
	}

	edges := make([]ScanEdge, 0)
	for pkg, imports := range pkgImports {
		srcTitle := pkgToTitle(pkg, modulePath)
		for dep := range imports {
			tgtTitle := pkgToTitle(dep, modulePath)
			edges = append(edges, ScanEdge{
				SourceTitle: srcTitle,
				TargetTitle: tgtTitle,
				EdgeType:    "depends_on",
			})
		}
	}

	return &ScanResult{
		Mode:  "merge",
		Nodes: nodes,
		Edges: edges,
	}, nil
}

// RunPlugin runs an external scanner plugin.
// The plugin must accept --path <dir> and write ImportGraphRequest-compatible JSON to stdout.
func RunPlugin(pluginPath string, projectPath string, maxFiles int) (*ScanResult, error) {
	absPath, err := filepath.Abs(pluginPath)
	if err != nil {
		return nil, fmt.Errorf("invalid plugin path: %w", err)
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("plugin not found: %w", err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("plugin path is a directory, not an executable: %s", absPath)
	}
	if info.Mode()&0111 == 0 {
		return nil, fmt.Errorf("plugin is not executable: %s", absPath)
	}

	args := []string{"--path", projectPath}
	if maxFiles > 0 {
		args = append(args, "--max-files", fmt.Sprintf("%d", maxFiles))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, absPath, args...)
	cmd.Stderr = os.Stderr

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("plugin exited with code %d", exitErr.ExitCode())
		}
		return nil, fmt.Errorf("failed to run plugin: %w", err)
	}

	var result ScanResult
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("plugin output is not valid JSON: %w", err)
	}

	if result.Mode == "" {
		result.Mode = "merge"
	}

	return &result, nil
}

func pkgToTitle(pkg, modulePath string) string {
	if pkg == modulePath {
		return "."
	}
	return strings.TrimPrefix(pkg, modulePath+"/")
}

func parseModulePath(gomod string) string {
	for _, line := range strings.Split(gomod, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			// Handle: module github.com/foo // comment
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return strings.Trim(parts[1], "\"")
			}
		}
	}
	return ""
}
