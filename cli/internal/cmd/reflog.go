package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/thask/cli/internal/telemetry"
)

var (
	reflogLimit       int
	reflogSince       string
	reflogSource      string
	reflogErrorsOnly  bool
	reflogShow        string
)

var reflogCmd = &cobra.Command{
	Use:     "reflog [<id>|search <query>]",
	Aliases: []string{"history"},
	Short:   "Show recent CLI invocations from ~/.thask/events.jsonl",
	Long: `By default lists the most recent invocations.
Use --show <id> for full detail, or 'reflog search <query>' for full-text grep.

Also available as 'thask history' (alias).`,
	Args: cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if reflogShow != "" {
			return runReflogShow(reflogShow)
		}
		if len(args) >= 1 && args[0] == "search" {
			if len(args) < 2 {
				return fmt.Errorf("usage: thask reflog search <query>")
			}
			return runReflogSearch(args[1])
		}
		return runReflogList()
	},
}

func runReflogList() error {
	s := telemetry.LoadState()
	limit := reflogLimit
	if limit <= 0 {
		limit = 20
	}
	cutoff := int64(0)
	if reflogSince != "" {
		if t, err := parseRelative(reflogSince); err == nil {
			cutoff = t.UnixMilli()
		}
	}
	type row struct {
		ts        int64
		ok        bool
		errClass  string
		cmd       string
		dur       int64
		source    string
		cliVer    string
	}
	all := make([]row, 0, 256)
	_, err := telemetry.Scan(s, func(e *telemetry.Event, _ []byte) {
		if e.Kind != telemetry.KindInvocation {
			return
		}
		if e.Ts < cutoff {
			return
		}
		if reflogSource != "" && e.Source != reflogSource {
			return
		}
		if reflogErrorsOnly && e.OK {
			return
		}
		all = append(all, row{e.Ts, e.OK, e.ErrorClass, e.Cmd, e.DurationMs, e.Source, e.CLIVersion})
	})
	if err != nil {
		return err
	}
	// Most recent first
	for i, j := 0, len(all)-1; i < j; i, j = i+1, j-1 {
		all[i], all[j] = all[j], all[i]
	}
	if len(all) > limit {
		all = all[:limit]
	}
	if len(all) == 0 {
		fmt.Println("(no matching invocations)")
		return nil
	}
	for _, r := range all {
		ts := time.UnixMilli(r.ts).Format("2006-01-02 15:04:05")
		mark := "✓"
		if !r.ok {
			mark = "✗"
		}
		extra := ""
		if r.errClass != "" {
			extra = "  [" + r.errClass + "]"
		}
		fmt.Printf("%s  %s  %s  %dms%s\n", ts, mark, r.cmd, r.dur, extra)
	}
	return nil
}

func runReflogShow(id string) error {
	s := telemetry.LoadState()
	var found []byte
	_, err := telemetry.Scan(s, func(e *telemetry.Event, raw []byte) {
		if found != nil {
			return
		}
		if strings.HasPrefix(e.ID, id) || strings.HasPrefix(e.Parent, id) {
			found = make([]byte, len(raw))
			copy(found, raw)
		}
	})
	if err != nil {
		return err
	}
	if found == nil {
		return fmt.Errorf("no event matching id prefix %q", id)
	}
	var pretty map[string]any
	if json.Unmarshal(found, &pretty) == nil {
		out, _ := json.MarshalIndent(pretty, "", "  ")
		fmt.Println(string(out))
		return nil
	}
	fmt.Println(string(found))
	return nil
}

func runReflogSearch(query string) error {
	if query == "" {
		return fmt.Errorf("empty query")
	}
	s := telemetry.LoadState()
	needle := []byte(query)
	var matches [][]byte
	_, err := telemetry.Scan(s, func(_ *telemetry.Event, raw []byte) {
		if bytes.Contains(raw, needle) {
			cp := make([]byte, len(raw))
			copy(cp, raw)
			matches = append(matches, cp)
		}
	})
	if err != nil {
		return err
	}
	if len(matches) == 0 {
		fmt.Println("(no matches)")
		return nil
	}
	fmt.Printf("%d match(es):\n", len(matches))
	for _, m := range matches {
		fmt.Println(string(m))
	}
	return nil
}

func init() {
	reflogCmd.Flags().IntVarP(&reflogLimit, "limit", "n", 20, "Max rows to show")
	reflogCmd.Flags().StringVar(&reflogSince, "since", "", "Window start (\"30 days ago\")")
	reflogCmd.Flags().StringVar(&reflogSource, "source", "", "Filter by source (terminal|mcp)")
	reflogCmd.Flags().BoolVar(&reflogErrorsOnly, "errors-only", false, "Only show failed invocations")
	reflogCmd.Flags().StringVar(&reflogShow, "show", "", "Show full detail for invocation id (prefix match OK)")
}
