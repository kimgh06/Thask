package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

func defaultInstallDir() string {
	switch runtime.GOOS {
	case "windows":
		// %LOCALAPPDATA%\thask
		local := os.Getenv("LOCALAPPDATA")
		if local == "" {
			home, _ := os.UserHomeDir()
			local = filepath.Join(home, "AppData", "Local")
		}
		return filepath.Join(local, "thask")
	default:
		// macOS, Linux
		return "/usr/local/bin"
	}
}

func binaryName() string {
	if runtime.GOOS == "windows" {
		return "thask.exe"
	}
	return "thask"
}

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install thask binary to system PATH",
	Long: fmt.Sprintf(`Install thask binary to a directory in your PATH.

Default install directory:
  macOS/Linux: /usr/local/bin  (may require sudo)
  Windows:     %%LOCALAPPDATA%%\thask

Examples:
  thask install                      # default location
  thask install --dir ~/.local/bin   # custom directory (Linux/macOS)
  sudo thask install                 # if permission denied (Linux/macOS)
  thask install --dir C:\tools       # custom directory (Windows)

Current OS: %s/%s`, runtime.GOOS, runtime.GOARCH),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, _ := cmd.Flags().GetString("dir")
		if dir == "" {
			dir = defaultInstallDir()
		}

		// Expand ~
		if strings.HasPrefix(dir, "~") {
			home, _ := os.UserHomeDir()
			dir = filepath.Join(home, dir[1:])
		}

		selfPath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to find current binary: %w", err)
		}
		selfPath, err = filepath.EvalSymlinks(selfPath)
		if err != nil {
			return fmt.Errorf("failed to resolve binary path: %w", err)
		}

		destPath := filepath.Join(dir, binaryName())

		if selfPath == destPath {
			fmt.Println("Already installed at " + destPath)
			return nil
		}

		// Ensure target dir exists
		if err := os.MkdirAll(dir, 0755); err != nil {
			if runtime.GOOS == "windows" {
				return fmt.Errorf("failed to create %s: %w", dir, err)
			}
			return fmt.Errorf("failed to create %s: %w\nTry: sudo thask install", dir, err)
		}

		input, err := os.ReadFile(selfPath)
		if err != nil {
			return fmt.Errorf("failed to read binary: %w", err)
		}

		if err := os.WriteFile(destPath, input, 0755); err != nil {
			if runtime.GOOS == "windows" {
				return fmt.Errorf("failed to write to %s: %w\nTry running as Administrator", destPath, err)
			}
			return fmt.Errorf("failed to write to %s: %w\nTry: sudo thask install", destPath, err)
		}

		fmt.Printf("Installed thask to %s\n", destPath)

		// Verify it's in PATH
		found, _ := exec.LookPath(binaryName())
		if found != "" {
			fmt.Printf("Verified: %s\n", found)
		} else {
			fmt.Printf("\nWarning: %s is not in your PATH.\n", dir)
			printPathHelp(dir)
		}

		return nil
	},
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove thask from system PATH",
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := exec.LookPath(binaryName())
		if err != nil {
			fmt.Println("thask is not installed in PATH")
			return nil
		}

		if err := os.Remove(path); err != nil {
			if runtime.GOOS == "windows" {
				return fmt.Errorf("failed to remove %s: %w\nTry running as Administrator", path, err)
			}
			return fmt.Errorf("failed to remove %s: %w\nTry: sudo thask uninstall", path, err)
		}

		fmt.Printf("Removed %s\n", path)
		return nil
	},
}

func printPathHelp(dir string) {
	switch runtime.GOOS {
	case "darwin":
		fmt.Printf("Add to ~/.zshrc:\n  export PATH=\"%s:$PATH\"\n", dir)
	case "linux":
		shell := os.Getenv("SHELL")
		if strings.Contains(shell, "zsh") {
			fmt.Printf("Add to ~/.zshrc:\n  export PATH=\"%s:$PATH\"\n", dir)
		} else {
			fmt.Printf("Add to ~/.bashrc:\n  export PATH=\"%s:$PATH\"\n", dir)
		}
	case "windows":
		fmt.Printf("Add to PATH via PowerShell (run as Administrator):\n")
		fmt.Printf("  [Environment]::SetEnvironmentVariable(\"PATH\", $env:PATH + \";%s\", \"User\")\n", dir)
		fmt.Printf("\nOr: Settings → System → About → Advanced → Environment Variables → PATH → Edit → Add: %s\n", dir)
	}
}

func init() {
	installCmd.Flags().String("dir", "", fmt.Sprintf("Target directory (default: %s)", defaultInstallDir()))

	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(uninstallCmd)
}
