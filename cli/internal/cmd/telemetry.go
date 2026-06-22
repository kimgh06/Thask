package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/thask/cli/internal/telemetry"
)

var telemetryCmd = &cobra.Command{
	Use:   "telemetry",
	Short: "Manage local-first CLI usage telemetry",
}

var telemetryStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show telemetry collection + payload + opt-in state",
	RunE: func(cmd *cobra.Command, args []string) error {
		s := telemetry.LoadState()
		dir := s.Dir
		info, _ := os.Stat(s.EventsPath())
		var events string
		if info == nil {
			events = "(not yet written)"
		} else {
			events = fmt.Sprintf("%d bytes, last write %s",
				info.Size(), info.ModTime().Format(time.RFC3339))
		}
		state := "enabled"
		if s.Disabled {
			state = "disabled (tombstone present)"
		}
		payloadDir := filepath.Join(dir, "payloads")
		blobCount := 0
		if entries, err := os.ReadDir(payloadDir); err == nil {
			blobCount = len(entries)
		}
		fmt.Printf("COLLECTION    %s\n", state)
		fmt.Printf("  meta        ✓ events.jsonl  (%s)\n", events)
		if s.CapturePayloads {
			fmt.Printf("  payloads    ✓ opt-in ON   (%d blobs in %s)\n", blobCount, payloadDir)
		} else {
			fmt.Printf("  payloads    ✗ opt-in OFF  (default — bodies never written)\n")
		}
		fmt.Printf("UPLOAD        ✗ not yet implemented (Phase 14)\n")
		fmt.Printf("INSTALL ID    %s\n", s.InstallID)
		fmt.Printf("DATA DIR      %s\n", dir)
		fmt.Println()
		fmt.Println("WHAT IS STORED LOCALLY (your machine only):")
		fmt.Println("  ✓ command name, duration, exit code, error class")
		fmt.Println("  ✓ os/arch/cli_version")
		fmt.Println("  ✓ backend host hash (sha256[:16]) + backend version header")
		fmt.Println("  ✓ MCP tool name + payload size buckets")
		fmt.Println("  ✗ never: raw URLs, tokens, node content, PII")
		if s.CapturePayloads {
			fmt.Println("  ⚠ payloads opt-in: request/response bodies stored under payloads/")
		}
		fmt.Println()
		fmt.Println("Run 'thask telemetry show schema' for the full field list.")
		return nil
	},
}

var telemetryShowCmd = &cobra.Command{
	Use:   "show <schema|last>",
	Short: "Print the event schema or the most recent raw event line",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "schema":
			fmt.Println(telemetrySchemaText)
		case "last":
			return showLastEvent()
		default:
			return fmt.Errorf("unknown subject %q; want schema or last", args[0])
		}
		return nil
	},
}

func showLastEvent() error {
	s := telemetry.LoadState()
	f, err := os.Open(s.EventsPath())
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("(no events recorded yet)")
			return nil
		}
		return err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	var last string
	for scanner.Scan() {
		if line := strings.TrimSpace(scanner.Text()); line != "" {
			last = line
		}
	}
	if last == "" {
		fmt.Println("(no events recorded yet)")
		return nil
	}
	// Pretty-print for readability — content unchanged.
	var pretty map[string]any
	if err := json.Unmarshal([]byte(last), &pretty); err == nil {
		out, _ := json.MarshalIndent(pretty, "", "  ")
		fmt.Println(string(out))
		return nil
	}
	fmt.Println(last)
	return nil
}

var telemetryConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Set telemetry options",
}

var telemetryConfigSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a telemetry config key (currently only capture_payloads)",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key, val := args[0], args[1]
		s := telemetry.LoadState()
		switch key {
		case "capture_payloads":
			b, err := strconv.ParseBool(val)
			if err != nil {
				return fmt.Errorf("capture_payloads expects true|false, got %q", val)
			}
			s.SetCapturePayloads(b)
			fmt.Printf("capture_payloads = %v\n", b)
		default:
			return fmt.Errorf("unknown key %q (supported: capture_payloads)", key)
		}
		return nil
	},
}

var telemetryDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Stop collecting telemetry (drops tombstone marker)",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Reuse the in-flight session's State so flipping Disabled here
		// also suppresses the invocation event Finalize would emit for
		// the disable command itself — leaving exactly one config_change
		// row behind, per Phase 13 §F.
		s := telemetry.LoadState()
		if cur := telemetry.Current(); cur != nil && cur.State != nil {
			s = cur.State
		}
		if s.Disabled {
			fmt.Println("Telemetry already disabled.")
			return nil
		}
		telemetry.Disable(s)
		fmt.Println("Telemetry disabled. Existing data preserved at " + s.Dir)
		return nil
	},
}

var telemetryEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Resume telemetry collection",
	RunE: func(cmd *cobra.Command, args []string) error {
		s := telemetry.LoadState()
		telemetry.Enable(s)
		fmt.Println("Telemetry enabled.")
		return nil
	},
}

var (
	purgeBefore      string
	purgeAll         bool
	purgePayloadsOnly bool
)

var telemetryPurgeCmd = &cobra.Command{
	Use:   "purge",
	Short: "Rewrite events.jsonl and/or delete payload blobs",
	RunE: func(cmd *cobra.Command, args []string) error {
		s := telemetry.LoadState()
		if purgeAll {
			if err := os.Remove(s.EventsPath()); err != nil && !os.IsNotExist(err) {
				return err
			}
			payloadDir := filepath.Join(s.Dir, "payloads")
			os.RemoveAll(payloadDir)
			fmt.Println("All telemetry data purged.")
			return nil
		}
		if purgePayloadsOnly {
			payloadDir := filepath.Join(s.Dir, "payloads")
			cutoff := time.Now().Add(-14 * 24 * time.Hour)
			if purgeBefore != "" {
				t, err := parseRelative(purgeBefore)
				if err != nil {
					return fmt.Errorf("--before: %w", err)
				}
				cutoff = t
			}
			n := removeOldBlobs(payloadDir, cutoff)
			fmt.Printf("Removed %d payload blobs older than %s.\n", n, cutoff.Format(time.RFC3339))
			return nil
		}
		cutoff, err := parseRelative(purgeBefore)
		if err != nil {
			return fmt.Errorf("--before: %w", err)
		}
		kept, removed, err := rewriteEventsBefore(s, cutoff)
		if err != nil {
			return err
		}
		fmt.Printf("Kept %d events, removed %d (cutoff %s).\n",
			kept, removed, cutoff.Format(time.RFC3339))
		return nil
	},
}

// rewriteEventsBefore atomically rewrites events.jsonl, keeping rows with
// ts >= cutoff_ms. Uses temp file + os.Rename per Phase 13 §13-1.
func rewriteEventsBefore(s *telemetry.State, cutoff time.Time) (kept, removed int, err error) {
	src := s.EventsPath()
	in, err := os.Open(src)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, 0, nil
		}
		return 0, 0, err
	}
	defer in.Close()

	tmp, err := os.CreateTemp(s.Dir, "events-purge-*.jsonl")
	if err != nil {
		return 0, 0, err
	}
	tmpPath := tmp.Name()
	defer func() {
		tmp.Close()
		os.Remove(tmpPath) // no-op if already renamed
	}()

	cutoffMs := cutoff.UnixMilli()
	w := bufio.NewWriter(tmp)
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var e struct {
			Ts int64 `json:"ts"`
		}
		if json.Unmarshal(line, &e) != nil || e.Ts < cutoffMs {
			removed++
			continue
		}
		w.Write(line)
		w.WriteByte('\n')
		kept++
	}
	if err := w.Flush(); err != nil {
		return kept, removed, err
	}
	if err := tmp.Close(); err != nil {
		return kept, removed, err
	}
	if err := os.Chmod(tmpPath, 0600); err != nil {
		return kept, removed, err
	}
	if err := os.Rename(tmpPath, src); err != nil {
		return kept, removed, err
	}
	return kept, removed, nil
}

func removeOldBlobs(dir string, cutoff time.Time) int {
	n := 0
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			if os.Remove(filepath.Join(dir, e.Name())) == nil {
				n++
			}
		}
	}
	return n
}

// parseRelative parses inputs like "30 days ago", "1 week ago", "yesterday",
// or RFC3339 / "2006-01-02". Empty input defaults to 30 days ago.
//
// ponytail: handles the common cases the user actually types; not a full
// natural-language parser. Add more units when someone files an issue.
func parseRelative(s string) (time.Time, error) {
	now := time.Now()
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return now.Add(-30 * 24 * time.Hour), nil
	}
	if s == "yesterday" {
		return now.Add(-24 * time.Hour), nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, nil
	}
	// "N (day|days|week|weeks|month|months) ago"
	parts := strings.Fields(s)
	if len(parts) == 3 && parts[2] == "ago" {
		n, err := strconv.Atoi(parts[0])
		if err == nil && n >= 0 {
			unit := strings.TrimSuffix(parts[1], "s")
			switch unit {
			case "day":
				return now.Add(-time.Duration(n) * 24 * time.Hour), nil
			case "week":
				return now.Add(-time.Duration(n) * 7 * 24 * time.Hour), nil
			case "month":
				return now.AddDate(0, -n, 0), nil
			case "hour":
				return now.Add(-time.Duration(n) * time.Hour), nil
			}
		}
	}
	return time.Time{}, fmt.Errorf("could not parse %q (try \"30 days ago\" or 2026-06-01)", s)
}

const telemetrySchemaText = `Telemetry event schema (JSONL — one line per event)

Required on every line:
  id              ULID-like 32-char hex, unique
  ts              unix epoch ms
  install         anonymous install id
  kind            install | invocation | mcp_call | payload | config_change
  v               schema version per kind

invocation fields:
  cli_version, cmd, source, duration_ms, exit_code, ok, error_class,
  backend_host_hash, backend_version, http_status,
  format, flags, output_bytes_bucket

mcp_call fields:
  parent (= invocation.id), tool_name, mcp_client, agent_model,
  agent_session_hash, payload_in_bucket, payload_out_bucket

payload fields (opt-in only; local_only:true):
  parent, blob_kind, blob_size, blob_ref, argv, req_method, req_path

install fields:
  os, arch, install_source, shell, first_run_at

config_change fields:
  config_key, config_delta

Redaction applied at write time:
  --token, THASK_TOKEN, Authorization: Bearer, URL credentials, JWT,
  Cookie / Set-Cookie, thsk_-prefixed tokens.

What's NEVER recorded:
  raw URLs, token values, node titles/descriptions, PII, file paths
  beyond install-source detection.`

func init() {
	telemetryPurgeCmd.Flags().StringVar(&purgeBefore, "before", "", "Remove events older than (\"30 days ago\", 2026-06-01)")
	telemetryPurgeCmd.Flags().BoolVar(&purgeAll, "all", false, "Remove all events + payloads (destructive)")
	telemetryPurgeCmd.Flags().BoolVar(&purgePayloadsOnly, "payloads-only", false, "Only remove payload blobs, leave events.jsonl alone")

	telemetryConfigCmd.AddCommand(telemetryConfigSetCmd)

	telemetryCmd.AddCommand(telemetryStatusCmd)
	telemetryCmd.AddCommand(telemetryShowCmd)
	telemetryCmd.AddCommand(telemetryConfigCmd)
	telemetryCmd.AddCommand(telemetryDisableCmd)
	telemetryCmd.AddCommand(telemetryEnableCmd)
	telemetryCmd.AddCommand(telemetryPurgeCmd)
}
