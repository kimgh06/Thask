package scan

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
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
}

func Run(opts ScanOptions) (*ScanResult, error) {
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

	err = filepath.Walk(opts.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		// Skip vendor, testdata, hidden dirs, _ prefixed dirs
		if info.IsDir() {
			base := info.Name()
			if base == "vendor" || base == "testdata" || strings.HasPrefix(base, ".") || strings.HasPrefix(base, "_") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".go") || strings.HasSuffix(info.Name(), "_test.go") {
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
			return strings.TrimSpace(strings.TrimPrefix(line, "module"))
		}
	}
	return ""
}
