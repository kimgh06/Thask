package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const aliasBlock = `
# --- thask aliases (auto-generated) ---
alias t='thask'
alias tn='thask node'
alias tnls='thask node list --pretty'
alias tnc='thask node create'
alias tnu='thask node update'
alias tnd='thask node delete'
alias tng='thask node get'
alias te='thask edge'
alias tels='thask edge list --pretty'
alias tec='thask edge create'
alias ted='thask edge delete'
alias tt='thask team'
alias ttls='thask team list --pretty'
alias tp='thask project'
alias tpls='thask project list --pretty'
alias tg='thask graph get'
alias tim='thask impact'
# --- end thask aliases ---
`

var aliasesCmd = &cobra.Command{
	Use:   "aliases",
	Short: "Manage shell aliases for thask commands",
}

var aliasesShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Print shell aliases to stdout (pipe to your shell rc file)",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print(aliasBlock)
	},
}

var aliasesInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Append aliases to ~/.zshrc or ~/.bashrc",
	RunE: func(cmd *cobra.Command, args []string) error {
		rcFile := detectRCFile()
		if f, _ := cmd.Flags().GetString("file"); f != "" {
			rcFile = f
		}

		data, err := os.ReadFile(rcFile)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to read %s: %w", rcFile, err)
		}

		if strings.Contains(string(data), "# --- thask aliases") {
			fmt.Printf("Aliases already installed in %s\n", rcFile)
			fmt.Println("To update, run: thask aliases uninstall && thask aliases install")
			return nil
		}

		f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to open %s: %w", rcFile, err)
		}
		defer f.Close()

		if _, err := f.WriteString(aliasBlock); err != nil {
			return fmt.Errorf("failed to write: %w", err)
		}

		fmt.Printf("Aliases installed in %s\n", rcFile)
		fmt.Println("Run: source " + rcFile)
		return nil
	},
}

var aliasesUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove thask aliases from shell rc file",
	RunE: func(cmd *cobra.Command, args []string) error {
		rcFile := detectRCFile()
		if f, _ := cmd.Flags().GetString("file"); f != "" {
			rcFile = f
		}

		data, err := os.ReadFile(rcFile)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", rcFile, err)
		}

		content := string(data)
		startMarker := "# --- thask aliases (auto-generated) ---"
		endMarker := "# --- end thask aliases ---"

		startIdx := strings.Index(content, startMarker)
		endIdx := strings.Index(content, endMarker)
		if startIdx == -1 || endIdx == -1 {
			fmt.Println("No thask aliases found in " + rcFile)
			return nil
		}

		cleaned := content[:startIdx] + content[endIdx+len(endMarker):]
		// Trim excess newlines
		cleaned = strings.TrimRight(cleaned, "\n") + "\n"

		if err := os.WriteFile(rcFile, []byte(cleaned), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", rcFile, err)
		}

		fmt.Printf("Aliases removed from %s\n", rcFile)
		return nil
	},
}

func detectRCFile() string {
	home, _ := os.UserHomeDir()

	// Check current shell
	shell := os.Getenv("SHELL")
	if strings.Contains(shell, "zsh") {
		return filepath.Join(home, ".zshrc")
	}
	if strings.Contains(shell, "bash") {
		return filepath.Join(home, ".bashrc")
	}

	// Check if zsh exists
	if _, err := exec.LookPath("zsh"); err == nil {
		return filepath.Join(home, ".zshrc")
	}

	return filepath.Join(home, ".bashrc")
}

func init() {
	aliasesInstallCmd.Flags().String("file", "", "Target rc file (default: auto-detect)")
	aliasesUninstallCmd.Flags().String("file", "", "Target rc file (default: auto-detect)")

	aliasesCmd.AddCommand(aliasesShowCmd)
	aliasesCmd.AddCommand(aliasesInstallCmd)
	aliasesCmd.AddCommand(aliasesUninstallCmd)
}
