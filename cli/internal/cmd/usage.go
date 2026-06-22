package cmd

import (
	"fmt"
	"sort"
	"time"

	"github.com/spf13/cobra"
	"github.com/thask/cli/internal/telemetry"
)

var (
	usageSince string
	usageLimit int
)

var usageCmd = &cobra.Command{
	Use:   "usage",
	Short: "Show local CLI usage stats from ~/.thask/events.jsonl",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUsageSummary(parseSinceOr30d())
	},
}

var usageTopCmd = &cobra.Command{
	Use:   "top",
	Short: "Top commands by invocation count",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUsageTop(parseSinceOr30d(), usageLimit)
	},
}

var usageErrorsCmd = &cobra.Command{
	Use:   "errors",
	Short: "Error funnel — counts grouped by error_class",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUsageErrors(parseSinceOr30d())
	},
}

func parseSinceOr30d() time.Time {
	if usageSince == "" {
		return time.Now().Add(-30 * 24 * time.Hour)
	}
	t, err := parseRelative(usageSince)
	if err != nil {
		// fall back to 30d window on parse failure rather than erroring out
		return time.Now().Add(-30 * 24 * time.Hour)
	}
	return t
}

func runUsageSummary(since time.Time) error {
	s := telemetry.LoadState()
	var invocations, errors int
	durations := make([]int64, 0, 256)
	byCmd := map[string]int{}
	byBackend := map[string]int{}
	bySource := map[string]int{}
	cutoff := since.UnixMilli()

	stats, err := telemetry.Scan(s, func(e *telemetry.Event, _ []byte) {
		if e.Kind != telemetry.KindInvocation || e.Ts < cutoff {
			return
		}
		invocations++
		if !e.OK {
			errors++
		}
		if e.DurationMs > 0 {
			durations = append(durations, e.DurationMs)
		}
		byCmd[e.Cmd]++
		if e.BackendHostHash != "" {
			byBackend[e.BackendHostHash]++
		}
		if e.Source != "" {
			bySource[e.Source]++
		}
	})
	if err != nil {
		return err
	}

	if invocations == 0 {
		fmt.Printf("No invocations recorded since %s.\n", since.Format("2006-01-02"))
		if stats.Skipped > 0 {
			fmt.Printf("(%d malformed lines skipped)\n", stats.Skipped)
		}
		return nil
	}
	p50, p95 := percentiles(durations)
	successPct := 100.0 * float64(invocations-errors) / float64(invocations)
	fmt.Printf("USAGE — since %s\n", since.Format("2006-01-02"))
	fmt.Println("───────────────────────────────────────────")
	fmt.Printf("Invocations    %d\n", invocations)
	fmt.Printf("Success rate   %.1f%%   (%d errors)\n", successPct, errors)
	fmt.Printf("Latency        p50 %dms   p95 %dms\n", p50, p95)
	if len(bySource) > 0 {
		var parts []string
		for k, v := range bySource {
			parts = append(parts, fmt.Sprintf("%s %d", k, v))
		}
		sort.Strings(parts)
		fmt.Printf("Sources        %s\n", joinComma(parts))
	}
	fmt.Println()
	fmt.Println("TOP COMMANDS")
	for _, kv := range topN(byCmd, 5) {
		fmt.Printf("  %4d  %s\n", kv.v, kv.k)
	}
	if len(byBackend) > 0 {
		fmt.Println()
		fmt.Println("BACKENDS")
		for _, kv := range topN(byBackend, 5) {
			fmt.Printf("  %4d  %s\n", kv.v, kv.k)
		}
	}
	if stats.Skipped > 0 {
		fmt.Printf("\n(%d malformed lines skipped)\n", stats.Skipped)
	}
	return nil
}

func runUsageTop(since time.Time, limit int) error {
	if limit <= 0 {
		limit = 10
	}
	s := telemetry.LoadState()
	counts := map[string]int{}
	sumDur := map[string]int64{}
	okCount := map[string]int{}
	cutoff := since.UnixMilli()
	_, err := telemetry.Scan(s, func(e *telemetry.Event, _ []byte) {
		if e.Kind != telemetry.KindInvocation || e.Ts < cutoff {
			return
		}
		counts[e.Cmd]++
		sumDur[e.Cmd] += e.DurationMs
		if e.OK {
			okCount[e.Cmd]++
		}
	})
	if err != nil {
		return err
	}
	if len(counts) == 0 {
		fmt.Println("No invocations in window.")
		return nil
	}
	rows := topN(counts, limit)
	fmt.Printf("%-30s %6s %8s %6s\n", "command", "count", "avg ms", "ok %")
	for _, r := range rows {
		avg := int64(0)
		if r.v > 0 {
			avg = sumDur[r.k] / int64(r.v)
		}
		okPct := 100.0 * float64(okCount[r.k]) / float64(r.v)
		fmt.Printf("%-30s %6d %8d %5.1f%%\n", r.k, r.v, avg, okPct)
	}
	return nil
}

func runUsageErrors(since time.Time) error {
	s := telemetry.LoadState()
	byClass := map[string]int{}
	var lastSeen = map[string]int64{}
	cutoff := since.UnixMilli()
	_, err := telemetry.Scan(s, func(e *telemetry.Event, _ []byte) {
		if e.Kind != telemetry.KindInvocation || e.Ts < cutoff || e.OK {
			return
		}
		class := e.ErrorClass
		if class == "" {
			class = "other"
		}
		byClass[class]++
		if e.Ts > lastSeen[class] {
			lastSeen[class] = e.Ts
		}
	})
	if err != nil {
		return err
	}
	if len(byClass) == 0 {
		fmt.Println("No errors in window. 🟢")
		return nil
	}
	fmt.Printf("ERRORS — since %s\n", since.Format("2006-01-02"))
	rows := topN(byClass, 20)
	for _, r := range rows {
		last := time.UnixMilli(lastSeen[r.k]).Format("2006-01-02 15:04")
		fmt.Printf("  %4d  %-10s   last: %s\n", r.v, r.k, last)
	}
	return nil
}

type kv struct {
	k string
	v int
}

func topN(m map[string]int, n int) []kv {
	out := make([]kv, 0, len(m))
	for k, v := range m {
		out = append(out, kv{k, v})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].v != out[j].v {
			return out[i].v > out[j].v
		}
		return out[i].k < out[j].k
	})
	if len(out) > n {
		out = out[:n]
	}
	return out
}

func percentiles(durations []int64) (p50, p95 int64) {
	if len(durations) == 0 {
		return 0, 0
	}
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	p50 = durations[len(durations)*50/100]
	p95Index := len(durations) * 95 / 100
	if p95Index >= len(durations) {
		p95Index = len(durations) - 1
	}
	p95 = durations[p95Index]
	return
}

func joinComma(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += ", "
		}
		out += p
	}
	return out
}

func init() {
	usageCmd.PersistentFlags().StringVar(&usageSince, "since", "", "Window start (\"30 days ago\", 2026-06-01)")
	usageTopCmd.Flags().IntVar(&usageLimit, "limit", 10, "Max rows")
	usageCmd.AddCommand(usageTopCmd)
	usageCmd.AddCommand(usageErrorsCmd)
}
