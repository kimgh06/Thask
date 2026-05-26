package cmd

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var selfUpdateCmd = &cobra.Command{
	Use:   "self-update",
	Short: "Download and install the latest Thask CLI from GitHub Releases",
	Long: `Fetches the latest published release, downloads the binary for your
platform, and atomically replaces the running thask binary in place.

Flags:
  --check      Only check; print whether an update is available, exit non-zero if so
  --version X  Install a specific version (e.g. --version 0.5.10) instead of latest`,
	RunE: runSelfUpdate,
}

func init() {
	selfUpdateCmd.Flags().Bool("check", false, "Check for updates without installing")
	selfUpdateCmd.Flags().String("version", "", "Install a specific version (default: latest)")
	rootCmd.AddCommand(selfUpdateCmd)
}

const (
	selfUpdateRepoAPI = "https://api.github.com/repos/kimgh06/Thask/releases"
	selfUpdateTimeout = 60 * time.Second
	// 200 MB ceiling caps a hostile / corrupted release from filling the
	// install directory. Real CLI binaries are well under this — a Linux
	// arm64 build is ~25 MB.
	selfUpdateMaxBytes = 200 * 1024 * 1024
)

func runSelfUpdate(cmd *cobra.Command, _ []string) error {
	checkOnly, _ := cmd.Flags().GetBool("check")
	pinned, _ := cmd.Flags().GetString("version")

	target := strings.TrimPrefix(pinned, "v")
	if target == "" {
		latest, err := fetchLatestVersion()
		if err != nil {
			return fmt.Errorf("fetch latest release: %w", err)
		}
		target = latest
	}

	current := Version
	if current == target {
		fmt.Fprintf(os.Stderr, "thask is already at v%s\n", current)
		return nil
	}
	fmt.Fprintf(os.Stderr, "current: v%s → target: v%s\n", current, target)
	if checkOnly {
		// Non-zero exit so shell scripts can detect "update available".
		os.Exit(1)
	}

	// Resolve current binary so we can replace it in place. os.Executable
	// returns the symlink-resolved path on macOS/Linux.
	binPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate current binary: %w", err)
	}
	binPath, _ = filepath.EvalSymlinks(binPath)

	assetURL, isTarball, err := platformAssetURL(target)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "downloading %s\n", assetURL)

	tmp, err := downloadToTemp(assetURL, filepath.Dir(binPath))
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer os.Remove(tmp)

	binaryFile := tmp
	if isTarball {
		extracted, xerr := extractTarballBinary(tmp, filepath.Dir(binPath))
		if xerr != nil {
			return fmt.Errorf("extract: %w", xerr)
		}
		defer os.Remove(extracted)
		binaryFile = extracted
	}

	if err := os.Chmod(binaryFile, 0o755); err != nil {
		return fmt.Errorf("chmod: %w", err)
	}
	// Rename across the same directory is atomic on POSIX, which guarantees the
	// running invocation never sees a half-written binary even if the user
	// re-runs thask mid-update.
	if err := os.Rename(binaryFile, binPath); err != nil {
		return fmt.Errorf("install to %s: %w (try sudo)", binPath, err)
	}

	fmt.Fprintf(os.Stderr, "✓ installed v%s at %s\n", target, binPath)
	return nil
}

func fetchLatestVersion() (string, error) {
	client := &http.Client{Timeout: selfUpdateTimeout}
	resp, err := client.Get(selfUpdateRepoAPI + "/latest")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	var rel struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", err
	}
	return strings.TrimPrefix(rel.TagName, "v"), nil
}

// platformAssetURL maps runtime.GOOS / GOARCH to the release artifact name
// the Thask release pipeline publishes. Unix platforms ship as tarballs;
// Windows ships as a raw .exe.
func platformAssetURL(version string) (url string, isTarball bool, err error) {
	key := runtime.GOOS + "-" + runtime.GOARCH
	switch key {
	case "darwin-arm64":
		return assetURL(version, "thask-darwin-arm64.tar.gz"), true, nil
	case "darwin-amd64":
		return assetURL(version, "thask-darwin-amd64.tar.gz"), true, nil
	case "linux-arm64":
		return assetURL(version, "thask-linux-arm64.tar.gz"), true, nil
	case "linux-amd64":
		return assetURL(version, "thask-linux-amd64.tar.gz"), true, nil
	case "windows-amd64":
		return assetURL(version, "thask-windows-amd64.exe"), false, nil
	}
	return "", false, fmt.Errorf("unsupported platform: %s", key)
}

func assetURL(version, name string) string {
	return fmt.Sprintf("https://github.com/kimgh06/Thask/releases/download/v%s/%s", version, name)
}

// downloadToTemp streams the URL into a temp file in `dir` (same directory as
// the binary, so the later os.Rename can be atomic). Handles redirects.
func downloadToTemp(url, dir string) (string, error) {
	client := &http.Client{Timeout: selfUpdateTimeout}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	f, err := os.CreateTemp(dir, "thask-download-*")
	if err != nil {
		return "", err
	}
	n, err := io.Copy(f, io.LimitReader(resp.Body, selfUpdateMaxBytes+1))
	if err == nil && n > selfUpdateMaxBytes {
		err = fmt.Errorf("download exceeded %d MB cap — aborted", selfUpdateMaxBytes>>20)
	}
	if err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", err
	}
	f.Close()
	return f.Name(), nil
}

// extractTarballBinary opens a .tar.gz, finds the single executable inside
// (the release tarballs contain just `thask`), and writes it to a temp file
// in `dir`. Returns the temp file path.
func extractTarballBinary(tarballPath, dir string) (string, error) {
	f, err := os.Open(tarballPath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return "", err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		// Be lenient about path layout — anything that looks like the thask
		// binary inside the tarball matches.
		base := filepath.Base(hdr.Name)
		if base != "thask" && !strings.HasPrefix(base, "thask-") {
			continue
		}
		out, err := os.CreateTemp(dir, "thask-extract-*")
		if err != nil {
			return "", err
		}
		n, err := io.Copy(out, io.LimitReader(tr, selfUpdateMaxBytes+1))
		if err == nil && n > selfUpdateMaxBytes {
			err = fmt.Errorf("tarball entry exceeded %d MB cap", selfUpdateMaxBytes>>20)
		}
		if err != nil {
			out.Close()
			os.Remove(out.Name())
			return "", err
		}
		out.Close()
		return out.Name(), nil
	}
	return "", fmt.Errorf("no thask binary inside tarball")
}
