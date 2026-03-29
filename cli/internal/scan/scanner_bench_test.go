package scan

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkScan1000Files(b *testing.B) {
	dir := b.TempDir()

	// Create go.mod
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module bench.test/project\n\ngo 1.22\n"), 0644)

	// Create 50 packages with 20 files each = 1000 files
	pkgs := make([]string, 50)
	for i := 0; i < 50; i++ {
		pkgName := fmt.Sprintf("pkg%03d", i)
		pkgs[i] = pkgName
		pkgDir := filepath.Join(dir, pkgName)
		os.MkdirAll(pkgDir, 0755)

		for j := 0; j < 20; j++ {
			var imports string
			// Each file imports 2-3 other packages
			if i > 0 {
				imports += fmt.Sprintf("\t\"bench.test/project/pkg%03d\"\n", i-1)
			}
			if i > 1 {
				imports += fmt.Sprintf("\t\"bench.test/project/pkg%03d\"\n", i-2)
			}

			content := fmt.Sprintf("package %s\n", pkgName)
			if imports != "" {
				content += fmt.Sprintf("\nimport (\n%s)\n", imports)
			}
			// Add a variable to make the imports used
			if i > 0 {
				content += fmt.Sprintf("\nvar _ = pkg%03d.Placeholder\n", i-1)
			}

			os.WriteFile(
				filepath.Join(pkgDir, fmt.Sprintf("file%02d.go", j)),
				[]byte(content),
				0644,
			)
		}

		// Add a Placeholder export
		os.WriteFile(
			filepath.Join(pkgDir, "export.go"),
			[]byte(fmt.Sprintf("package %s\n\nvar Placeholder = true\n", pkgName)),
			0644,
		)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Run(ScanOptions{Path: dir, MaxFiles: 2000})
		if err != nil {
			b.Fatal(err)
		}
	}
}
