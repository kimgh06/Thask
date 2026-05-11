package scan

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// tsScanner scans TypeScript/JavaScript projects for intra-project import dependencies.
// Nodes are directories (mirroring the Go scanner's package granularity).
type tsScanner struct{}

func (tsScanner) Name() string { return "ts" }

var tsExts = []string{".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs"}

// Resolution attempt order for a relative import path (without trailing slash).
var tsResolveSuffixes = []string{
	"",
	".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs",
	"/index.ts", "/index.tsx", "/index.js", "/index.jsx",
}

var (
	// import { ... } from 'x'  /  import x from 'x'  (multi-line friendly)
	tsImportFromRE = regexp.MustCompile(`import\s+(?:[^'"\n;]|\n)*?from\s+['"]([^'"]+)['"]`)
	// import 'x'   (side-effect)
	tsSideEffectImportRE = regexp.MustCompile(`import\s+['"]([^'"]+)['"]`)
	// require('x')
	tsRequireRE = regexp.MustCompile(`require\(\s*['"]([^'"]+)['"]\s*\)`)
	// import('x')  (dynamic)
	tsDynImportRE = regexp.MustCompile(`import\(\s*['"]([^'"]+)['"]\s*\)`)
	// export * from 'x'   /   export { a } from 'x'  (multi-line friendly)
	tsExportFromRE = regexp.MustCompile(`export\s+(?:\*|\{(?:[^}]|\n)*\})\s+from\s+['"]([^'"]+)['"]`)
)

var tsSkipDirs = map[string]bool{
	"node_modules": true,
	"dist":         true,
	"build":        true,
	".next":        true,
	".svelte-kit":  true,
	".nuxt":        true,
	"out":          true,
	"coverage":     true,
	".turbo":       true,
	"vendor":       true,
}

func tsIsSkippedDir(name string) bool {
	if tsSkipDirs[name] {
		return true
	}
	if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") {
		return true
	}
	return false
}

func tsIsSourceFile(name string) bool {
	// Exclude tests + declarations
	if strings.HasSuffix(name, ".d.ts") {
		return false
	}
	for _, suf := range []string{".test.ts", ".test.tsx", ".test.js", ".test.jsx",
		".spec.ts", ".spec.tsx", ".spec.js", ".spec.jsx"} {
		if strings.HasSuffix(name, suf) {
			return false
		}
	}
	for _, ext := range tsExts {
		if strings.HasSuffix(name, ext) {
			return true
		}
	}
	return false
}

// resolveTSImport attempts to resolve a relative import spec against importerFile's directory.
// Returns the absolute path to the resolved file, or "" if unresolved.
func resolveTSImport(importerFile, spec string) string {
	if !strings.HasPrefix(spec, "./") && !strings.HasPrefix(spec, "../") {
		return ""
	}
	base := filepath.Dir(importerFile)
	candidate := filepath.Clean(filepath.Join(base, spec))
	return tsLookupFile(candidate)
}

// tsLookupFile tries the canonical TS/JS file-resolution suffixes for an absolute path.
func tsLookupFile(candidate string) string {
	for _, suf := range tsResolveSuffixes {
		p := candidate + suf
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return p
		}
	}
	return ""
}

// ---------------------------------------------------------------------------
// tsconfig.json / jsconfig.json path-alias support
// ---------------------------------------------------------------------------

type tsAliasRule struct {
	pattern     string   // e.g. "$lib/*" or "$lib"
	targets     []string // relative-to-baseDir, e.g. ["src/lib/*"]
	hasWildcard bool
}

type tsAliases struct {
	baseDir string // absolute path to baseUrl
	rootDir string // absolute scan root; alias targets outside this are ignored
	rules   []tsAliasRule
	// ignored is a set of patterns that match but should NOT produce edges
	// (e.g. SvelteKit's $app/* runtime modules).
	ignored []tsAliasRule
}

// Pre-compiled regex for stripping JSON trailing commas.
var tsJSONTrailingCommaRE = regexp.MustCompile(`,(\s*[}\]])`)

// stripCommentsStringAware removes // line comments and /* */ block comments
// from src, preserving any "//" or "/*" appearing inside double-quoted string
// literals. Used by both tsconfig parsing (where "https://..." must not be
// truncated) and the TS/JS import extractor.
func stripCommentsStringAware(src string) string {
	var out strings.Builder
	out.Grow(len(src))
	n := len(src)
	inString := false
	inLine := false
	inBlock := false
	for i := 0; i < n; i++ {
		c := src[i]
		if inLine {
			if c == '\n' {
				inLine = false
				out.WriteByte(c)
			}
			continue
		}
		if inBlock {
			if c == '*' && i+1 < n && src[i+1] == '/' {
				inBlock = false
				i++
			}
			continue
		}
		if inString {
			out.WriteByte(c)
			if c == '\\' && i+1 < n {
				out.WriteByte(src[i+1])
				i++
				continue
			}
			if c == '"' {
				inString = false
			}
			continue
		}
		if c == '"' {
			inString = true
			out.WriteByte(c)
			continue
		}
		if c == '/' && i+1 < n {
			if src[i+1] == '/' {
				inLine = true
				i++
				continue
			}
			if src[i+1] == '*' {
				inBlock = true
				i++
				continue
			}
		}
		out.WriteByte(c)
	}
	return out.String()
}

// stripJSONComments removes // line comments, /* */ block comments, and trailing commas.
// String-aware: "//" or "/*" inside string literals (e.g. "https://...") are preserved.
func stripJSONComments(b []byte) []byte {
	stripped := stripCommentsStringAware(string(b))
	return []byte(tsJSONTrailingCommaRE.ReplaceAllString(stripped, "$1"))
}

type tsConfigRaw struct {
	Extends         string `json:"extends"`
	CompilerOptions struct {
		BaseURL string              `json:"baseUrl"`
		Paths   map[string][]string `json:"paths"`
	} `json:"compilerOptions"`
}

func loadTSAliases(root string) *tsAliases {
	rootClean := filepath.Clean(root)
	a := &tsAliases{baseDir: rootClean, rootDir: rootClean}

	// Try tsconfig.json then jsconfig.json.
	var raw *tsConfigRaw
	var cfgPath string
	for _, name := range []string{"tsconfig.json", "jsconfig.json"} {
		p := filepath.Join(root, name)
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var parsed tsConfigRaw
		if jerr := json.Unmarshal(stripJSONComments(data), &parsed); jerr != nil {
			continue
		}
		raw = &parsed
		cfgPath = p
		break
	}

	if raw != nil {
		baseURL := raw.CompilerOptions.BaseURL
		if baseURL == "" {
			baseURL = "."
		}
		a.baseDir = filepath.Clean(filepath.Join(filepath.Dir(cfgPath), baseURL))
		for pat, targets := range raw.CompilerOptions.Paths {
			rule := tsAliasRule{
				pattern:     pat,
				targets:     targets,
				hasWildcard: strings.HasSuffix(pat, "/*"),
			}
			a.rules = append(a.rules, rule)
		}
	}

	// SvelteKit auto-detection: add $lib/* -> src/lib/* if not already present,
	// and mark $app/* as ignored (runtime-only).
	isSvelteKit := false
	if raw != nil && strings.Contains(raw.Extends, ".svelte-kit") {
		isSvelteKit = true
	}
	if _, err := os.Stat(filepath.Join(root, ".svelte-kit")); err == nil {
		isSvelteKit = true
	}
	if _, err := os.Stat(filepath.Join(root, "svelte.config.js")); err == nil {
		isSvelteKit = true
	}
	if isSvelteKit {
		if !a.hasPattern("$lib/*") && !a.hasPattern("$lib") {
			a.rules = append(a.rules,
				tsAliasRule{pattern: "$lib/*", targets: []string{"src/lib/*"}, hasWildcard: true},
				tsAliasRule{pattern: "$lib", targets: []string{"src/lib"}, hasWildcard: false},
			)
		}
		a.ignored = append(a.ignored,
			tsAliasRule{pattern: "$app/*", hasWildcard: true},
			tsAliasRule{pattern: "$env/*", hasWildcard: true},
			tsAliasRule{pattern: "$service-worker", hasWildcard: false},
		)
	}

	return a
}

func (a *tsAliases) hasPattern(p string) bool {
	for _, r := range a.rules {
		if r.pattern == p {
			return true
		}
	}
	return false
}

// matchRule returns (suffix, true) if spec matches the rule, where suffix is the
// portion that replaces the trailing "*" (or "" for an exact match).
func matchRule(spec string, rule tsAliasRule) (string, bool) {
	if rule.hasWildcard {
		prefix := strings.TrimSuffix(rule.pattern, "*") // keeps trailing "/"
		if strings.HasPrefix(spec, prefix) {
			return spec[len(prefix):], true
		}
		return "", false
	}
	if spec == rule.pattern {
		return "", true
	}
	return "", false
}

// Resolve attempts to map an alias-ish import spec to an absolute file path.
// Returns ("", false) if no rule matches. Returns ("", true) if the spec matches
// an explicitly-ignored alias (e.g. $app/navigation) so callers can skip silently.
func (a *tsAliases) Resolve(spec string) (resolved string, matched bool) {
	if a == nil {
		return "", false
	}
	for _, rule := range a.ignored {
		if _, ok := matchRule(spec, rule); ok {
			return "", true // matched but intentionally unresolved
		}
	}
	for _, rule := range a.rules {
		suffix, ok := matchRule(spec, rule)
		if !ok {
			continue
		}
		for _, tgt := range rule.targets {
			expanded := tgt
			if rule.hasWildcard {
				expanded = strings.TrimSuffix(tgt, "*") + suffix
			}
			abs := filepath.Clean(filepath.Join(a.baseDir, expanded))
			// Guard: ignore targets that escape the scan root.
			if a.rootDir != "" {
				rootClean := filepath.Clean(a.rootDir)
				if abs != rootClean && !strings.HasPrefix(abs+string(filepath.Separator), rootClean+string(filepath.Separator)) {
					continue
				}
			}
			if p := tsLookupFile(abs); p != "" {
				return p, true
			}
			// Allow targets pointing directly at existing files.
			if info, err := os.Stat(abs); err == nil && !info.IsDir() {
				return abs, true
			}
		}
		return "", true // pattern matched but file not found — still "alias-known"
	}
	return "", false
}

// extractTSImports returns the list of import specs found in the source content.
// Processes the whole file at once (multi-line import friendly) with string-aware
// comment stripping (preserves "//" inside string literals).
//
// Known limitation: import-like substrings inside string literals
// (e.g. `const code = "import './fake'"`) may produce false positives because
// the import regexes themselves are not string-aware. A full TS parser would be
// required to handle this; out of scope for v1.
func extractTSImports(content string) []string {
	stripped := stripCommentsStringAware(content)

	seen := make(map[string]bool)
	var specs []string
	regexes := []*regexp.Regexp{
		tsImportFromRE,
		tsExportFromRE,
		tsRequireRE,
		tsDynImportRE,
		tsSideEffectImportRE,
	}
	for _, re := range regexes {
		for _, m := range re.FindAllStringSubmatch(stripped, -1) {
			if len(m) >= 2 && !seen[m[1]] {
				seen[m[1]] = true
				specs = append(specs, m[1])
			}
		}
	}
	return specs
}

func (tsScanner) Scan(opts ScanOptions) (*ScanResult, error) {
	if opts.MaxFiles <= 0 {
		opts.MaxFiles = 500
	}

	root := opts.Path
	aliases := loadTSAliases(root)

	// Walk files: collect (file -> dir, dir set, file imports)
	type fileEntry struct {
		path string
		dir  string // relative dir from root, "." for root
	}
	var files []fileEntry
	fileCount := 0

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		// Skip symlinks (both file and directory symlinks); we do not follow them
		// to avoid cycles and accidental traversal outside the scan root.
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		if d.IsDir() {
			if path == root {
				return nil
			}
			if tsIsSkippedDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !tsIsSourceFile(d.Name()) {
			return nil
		}
		fileCount++
		if fileCount > opts.MaxFiles {
			return fmt.Errorf("exceeded maxFiles limit (%d/%d files found)", fileCount, opts.MaxFiles)
		}
		relDir, _ := filepath.Rel(root, filepath.Dir(path))
		if relDir == "" {
			relDir = "."
		}
		relDir = filepath.ToSlash(relDir)
		files = append(files, fileEntry{path: path, dir: relDir})
		return nil
	})
	if err != nil {
		return nil, err
	}

	// dirSet collects every directory containing source files.
	dirSet := make(map[string]bool)
	for _, f := range files {
		dirSet[f.dir] = true
	}

	// Edges: source dir -> target dir, deduped.
	edgeSet := make(map[string]bool) // "src|tgt"
	edges := make([]ScanEdge, 0)

	for _, f := range files {
		data, readErr := os.ReadFile(f.path)
		if readErr != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping %s: %v\n", f.path, readErr)
			continue
		}
		specs := extractTSImports(string(data))
		for _, spec := range specs {
			resolved := resolveTSImport(f.path, spec)
			if resolved == "" {
				if aliasPath, matched := aliases.Resolve(spec); matched {
					if aliasPath == "" {
						continue // alias known but unresolved / intentionally ignored
					}
					resolved = aliasPath
				} else {
					continue
				}
			}
			tgtDir, _ := filepath.Rel(root, filepath.Dir(resolved))
			if tgtDir == "" {
				tgtDir = "."
			}
			tgtDir = filepath.ToSlash(tgtDir)
			if tgtDir == f.dir {
				continue
			}
			// Ensure target dir is also a node (it should be, since resolved is a source file).
			dirSet[tgtDir] = true
			key := f.dir + "|" + tgtDir
			if edgeSet[key] {
				continue
			}
			edgeSet[key] = true
			edges = append(edges, ScanEdge{
				SourceTitle: f.dir,
				TargetTitle: tgtDir,
				EdgeType:    "depends_on",
			})
		}
	}

	// Sorted directory list for stable layout.
	dirs := make([]string, 0, len(dirSet))
	for d := range dirSet {
		dirs = append(dirs, d)
	}
	sort.Strings(dirs)

	nodes := make([]ScanNode, 0, len(dirs))
	for i, d := range dirs {
		nodes = append(nodes, ScanNode{
			Type:      "TASK",
			Title:     d,
			Status:    "IN_PROGRESS",
			PositionX: float64(i%5) * 200,
			PositionY: float64(i/5) * 150,
		})
	}

	return &ScanResult{
		Mode:  "merge",
		Nodes: nodes,
		Edges: edges,
	}, nil
}
