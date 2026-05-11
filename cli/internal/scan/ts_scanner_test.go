package scan

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func hasEdge(edges []ScanEdge, src, tgt string) bool {
	for _, e := range edges {
		if e.SourceTitle == src && e.TargetTitle == tgt {
			return true
		}
	}
	return false
}

func hasNode(nodes []ScanNode, title string) bool {
	for _, n := range nodes {
		if n.Title == title {
			return true
		}
	}
	return false
}

func TestTSScanFixture(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "package.json"), `{"name":"x","version":"0.0.0"}`)
	writeFile(t, filepath.Join(dir, "src", "index.ts"), `
import { foo } from './foo';
import { bar } from './bar';
import React from 'react';
console.log(foo, bar, React);
`)
	writeFile(t, filepath.Join(dir, "src", "foo.ts"), `
import { bar } from './bar';
export const foo = bar;
`)
	writeFile(t, filepath.Join(dir, "src", "bar.ts"), `
export const bar = 1;
`)
	writeFile(t, filepath.Join(dir, "src", "utils", "helper.ts"), `
import { foo } from '../foo';
export const helper = foo;
`)

	result, err := Run(ScanOptions{Path: dir, Language: "ts"})
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	// Nodes: "src" and "src/utils"
	if !hasNode(result.Nodes, "src") {
		t.Errorf("missing node 'src'; got nodes=%+v", result.Nodes)
	}
	if !hasNode(result.Nodes, "src/utils") {
		t.Errorf("missing node 'src/utils'; got nodes=%+v", result.Nodes)
	}
	if len(result.Nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d: %+v", len(result.Nodes), result.Nodes)
	}

	// Edge: src/utils -> src (helper imports ../foo)
	if !hasEdge(result.Edges, "src/utils", "src") {
		t.Errorf("missing edge src/utils -> src; edges=%+v", result.Edges)
	}

	// Should NOT have intra-dir edges (src -> src) — all foo/bar imports are within "src".
	if hasEdge(result.Edges, "src", "src") {
		t.Errorf("unexpected self-edge src -> src")
	}

	if result.Mode != "merge" {
		t.Errorf("expected mode=merge, got %s", result.Mode)
	}
}

func TestTSScanIgnoresNodeModules(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"), `{"name":"x"}`)
	writeFile(t, filepath.Join(dir, "src", "a.ts"), `export const a = 1;`)
	writeFile(t, filepath.Join(dir, "node_modules", "pkg", "index.ts"), `export const x = 1;`)
	writeFile(t, filepath.Join(dir, "dist", "out.js"), `module.exports = {};`)

	result, err := Run(ScanOptions{Path: dir, Language: "ts"})
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	for _, n := range result.Nodes {
		if n.Title == "node_modules/pkg" || n.Title == "dist" {
			t.Errorf("scanner included excluded dir: %s", n.Title)
		}
	}
}

func TestTSScanAutoDetect(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"), `{"name":"x"}`)
	writeFile(t, filepath.Join(dir, "a.ts"), `export const a = 1;`)

	s := detectScanner(ScanOptions{Path: dir})
	if s.Name() != "ts" {
		t.Errorf("expected ts scanner via auto-detect, got %s", s.Name())
	}
}

func TestTSScanAutoDetectPrefersGo(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/x\n")
	writeFile(t, filepath.Join(dir, "package.json"), `{"name":"x"}`)
	s := detectScanner(ScanOptions{Path: dir})
	if s.Name() != "go" {
		t.Errorf("expected go scanner when go.mod present, got %s", s.Name())
	}
}

func TestTSScanIgnoresExternalImports(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"), `{"name":"x"}`)
	writeFile(t, filepath.Join(dir, "src", "a.ts"), `
import React from 'react';
import { foo } from '@scope/lib';
import { bar } from '$lib/utils';
export const a = 1;
`)
	writeFile(t, filepath.Join(dir, "src", "b.ts"), `export const b = 1;`)

	result, err := Run(ScanOptions{Path: dir, Language: "ts"})
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if len(result.Edges) != 0 {
		t.Errorf("expected 0 edges (all imports external), got %d: %+v", len(result.Edges), result.Edges)
	}
}

func TestTSScanIgnoresTestAndDts(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"), `{"name":"x"}`)
	writeFile(t, filepath.Join(dir, "src", "a.ts"), `export const a = 1;`)
	writeFile(t, filepath.Join(dir, "src", "a.test.ts"), `import './a'; test('x', () => {});`)
	writeFile(t, filepath.Join(dir, "src", "a.spec.ts"), `import './a';`)
	writeFile(t, filepath.Join(dir, "types", "a.d.ts"), `declare const x: number;`)

	result, err := Run(ScanOptions{Path: dir, Language: "ts"})
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	// Only "src" should be a node ("types" had only .d.ts which we exclude).
	for _, n := range result.Nodes {
		if n.Title == "types" {
			t.Errorf("scanner included 'types' dir (only .d.ts files)")
		}
	}
}

func TestTSScanResolvesTSConfigPaths(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"), `{"name":"x"}`)
	writeFile(t, filepath.Join(dir, "tsconfig.json"), `{
  "compilerOptions": {
    "baseUrl": ".",
    "paths": {
      "@/*": ["src/*"],
      "@utils": ["src/utils/index.ts"]
    }
  }
}`)
	writeFile(t, filepath.Join(dir, "src", "app", "main.ts"), `
import { foo } from '@/utils/foo';
import { x } from '@utils';
export const main = foo;
`)
	writeFile(t, filepath.Join(dir, "src", "utils", "foo.ts"), `export const foo = 1;`)
	writeFile(t, filepath.Join(dir, "src", "utils", "index.ts"), `export const x = 1;`)

	result, err := Run(ScanOptions{Path: dir, Language: "ts"})
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if !hasEdge(result.Edges, "src/app", "src/utils") {
		t.Errorf("missing alias-resolved edge src/app -> src/utils; edges=%+v", result.Edges)
	}
}

func TestTSScanSvelteKitLibAlias(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"), `{"name":"x"}`)
	writeFile(t, filepath.Join(dir, "svelte.config.js"), `export default {};`)
	writeFile(t, filepath.Join(dir, "src", "routes", "page.ts"), `
import { thing } from '$lib/thing';
export const p = thing;
`)
	writeFile(t, filepath.Join(dir, "src", "lib", "thing.ts"), `export const thing = 1;`)

	result, err := Run(ScanOptions{Path: dir, Language: "ts"})
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if !hasEdge(result.Edges, "src/routes", "src/lib") {
		t.Errorf("missing $lib alias edge src/routes -> src/lib; edges=%+v", result.Edges)
	}
}

func TestTSScanTSConfigWithComments(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"), `{"name":"x"}`)
	writeFile(t, filepath.Join(dir, "tsconfig.json"), `{
  // top-level comment
  "compilerOptions": {
    /* block comment */
    "baseUrl": ".",
    "paths": {
      "@/*": ["src/*"], // trailing line comment
    },
  },
}`)
	writeFile(t, filepath.Join(dir, "src", "app", "main.ts"), `
import { x } from '@/utils/foo';
export const m = x;
`)
	writeFile(t, filepath.Join(dir, "src", "utils", "foo.ts"), `export const x = 1;`)

	result, err := Run(ScanOptions{Path: dir, Language: "ts"})
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if !hasEdge(result.Edges, "src/app", "src/utils") {
		t.Errorf("tsconfig-with-comments alias did not resolve; edges=%+v", result.Edges)
	}
}

func TestTSScanIgnoresAppAlias(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"), `{"name":"x"}`)
	writeFile(t, filepath.Join(dir, "svelte.config.js"), `export default {};`)
	writeFile(t, filepath.Join(dir, "src", "routes", "page.ts"), `
import { goto } from '$app/navigation';
import { env } from '$env/dynamic/private';
export const p = goto;
`)
	writeFile(t, filepath.Join(dir, "src", "lib", "thing.ts"), `export const thing = 1;`)

	result, err := Run(ScanOptions{Path: dir, Language: "ts"})
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	for _, e := range result.Edges {
		if e.SourceTitle == "src/routes" {
			t.Errorf("$app/$env imports must not produce edges; got %+v", e)
		}
	}
}

func TestTSScanMultilineImport(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"), `{"name":"x"}`)
	writeFile(t, filepath.Join(dir, "a", "a.ts"), `
import {
  foo,
  bar,
} from '../b/baz';
export const a = foo;
`)
	writeFile(t, filepath.Join(dir, "b", "baz.ts"), `
export const foo = 1;
export const bar = 2;
`)

	result, err := Run(ScanOptions{Path: dir, Language: "ts"})
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if !hasEdge(result.Edges, "a", "b") {
		t.Errorf("missing multi-line import edge a -> b; edges=%+v", result.Edges)
	}
}

func TestTSScanIgnoresHTTPSInJSONString(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"), `{"name":"x"}`)
	writeFile(t, filepath.Join(dir, "tsconfig.json"), `{
  "compilerOptions": {
    "baseUrl": "https://cdn.example/",
    "paths": {
      "@/*": ["src/*"]
    }
  }
}`)
	writeFile(t, filepath.Join(dir, "src", "app", "main.ts"), `
import { x } from '@/utils/foo';
export const m = x;
`)
	writeFile(t, filepath.Join(dir, "src", "utils", "foo.ts"), `export const x = 1;`)

	result, err := Run(ScanOptions{Path: dir, Language: "ts"})
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	// baseUrl is "https://..." which is non-sensical for the filesystem; we
	// just want to ensure stripJSONComments didn't truncate it and parsing
	// didn't crash. Aliases targeting outside root should not produce edges.
	for _, e := range result.Edges {
		if e.SourceTitle == "src/app" {
			t.Errorf("unexpected edge from src/app via out-of-root alias: %+v", e)
		}
	}
}

func TestTSScanSideEffectImport(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"), `{"name":"x"}`)
	writeFile(t, filepath.Join(dir, "a", "a.ts"), `
import '../b/polyfills';
export const a = 1;
`)
	writeFile(t, filepath.Join(dir, "b", "polyfills.ts"), `globalThis.x = 1;`)

	result, err := Run(ScanOptions{Path: dir, Language: "ts"})
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if !hasEdge(result.Edges, "a", "b") {
		t.Errorf("missing side-effect import edge a -> b; edges=%+v", result.Edges)
	}
}

func TestTSScanAliasPathTraversalIgnored(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"), `{"name":"x"}`)
	writeFile(t, filepath.Join(dir, "tsconfig.json"), `{
  "compilerOptions": {
    "baseUrl": ".",
    "paths": {
      "@/*": ["../../etc/*"]
    }
  }
}`)
	writeFile(t, filepath.Join(dir, "src", "app", "main.ts"), `
import { x } from '@/passwd';
export const m = x;
`)

	result, err := Run(ScanOptions{Path: dir, Language: "ts"})
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	// Must not panic, must not produce edges resolving outside root.
	for _, e := range result.Edges {
		if e.SourceTitle == "src/app" {
			t.Errorf("alias root-escape must not produce edges; got %+v", e)
		}
	}
}

func TestTSScanIgnoresImportInStringLiteral(t *testing.T) {
	// Known limitation: import regexes are not string-literal-aware.
	// A full TS parser would be required to suppress this false positive.
	t.Skip("string-literal-aware import detection out of scope for v1")
}

func TestTSScanCommentedImportsIgnored(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"), `{"name":"x"}`)
	writeFile(t, filepath.Join(dir, "a", "a.ts"), `
// import { x } from '../b/b';
/* import { y } from '../b/b'; */
export const a = 1;
`)
	writeFile(t, filepath.Join(dir, "b", "b.ts"), `export const b = 1;`)

	result, err := Run(ScanOptions{Path: dir, Language: "ts"})
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if hasEdge(result.Edges, "a", "b") {
		t.Errorf("commented-out imports must not produce edges; got edges=%+v", result.Edges)
	}
}
