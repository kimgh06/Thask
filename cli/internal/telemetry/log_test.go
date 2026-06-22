package telemetry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newTempState(t *testing.T) *State {
	t.Helper()
	dir := t.TempDir()
	thaskDir := filepath.Join(dir, ".thask")
	if err := os.MkdirAll(thaskDir, 0700); err != nil {
		t.Fatal(err)
	}
	return &State{Dir: thaskDir, InstallID: "test_install"}
}

func TestIDIsUniqueAndTimeOrdered(t *testing.T) {
	a := NewIDAt(time.UnixMilli(1_700_000_000_000))
	b := NewIDAt(time.UnixMilli(1_700_000_001_000))
	if len(a) != 32 || len(b) != 32 {
		t.Fatalf("expected 32 hex chars, got %d/%d", len(a), len(b))
	}
	if a >= b {
		t.Errorf("expected earlier id %q to sort before later id %q", a, b)
	}
}

func TestLogAppendsLine(t *testing.T) {
	s := newTempState(t)
	Log(s, &Event{Kind: KindInvocation, Cmd: "test"})
	Log(s, &Event{Kind: KindInvocation, Cmd: "other"})

	data, err := os.ReadFile(s.EventsPath())
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %q", len(lines), string(data))
	}
	if !strings.Contains(lines[0], `"cmd":"test"`) {
		t.Errorf("line 0 missing cmd:test: %s", lines[0])
	}
}

func TestLogNoOpWhenDisabled(t *testing.T) {
	s := newTempState(t)
	s.Disabled = true
	Log(s, &Event{Kind: KindInvocation, Cmd: "must-not-appear"})
	if _, err := os.Stat(s.EventsPath()); !os.IsNotExist(err) {
		t.Errorf("events.jsonl should not exist when disabled, stat err: %v", err)
	}
}

func TestLogSurvivesMissingDir(t *testing.T) {
	// Point at a path the user has not created.
	s := &State{Dir: filepath.Join(t.TempDir(), "does-not-exist-yet")}
	Log(s, &Event{Kind: KindInvocation, Cmd: "auto-creates-dir"})
	if _, err := os.Stat(s.EventsPath()); err != nil {
		t.Errorf("expected events.jsonl auto-created, err: %v", err)
	}
}

func TestScanSkipsCorruptLines(t *testing.T) {
	s := newTempState(t)
	Log(s, &Event{Kind: KindInvocation, Cmd: "first"})
	// Append a corrupt line bypassing Log() to simulate manual edit.
	f, err := os.OpenFile(s.EventsPath(), os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString(`{"id":"x"` + "\n") // malformed JSON
	f.WriteString(`{"id":"y","ts":1,"install":"i","kind":"future_kind","v":99}` + "\n")
	f.Close()

	var seenKinds []string
	stats, err := Scan(s, func(e *Event, _ []byte) {
		seenKinds = append(seenKinds, e.Kind)
	})
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}
	if stats.Parsed != 2 {
		t.Errorf("expected Parsed=2, got %d", stats.Parsed)
	}
	if stats.Skipped != 1 {
		t.Errorf("expected Skipped=1, got %d", stats.Skipped)
	}
	if len(seenKinds) != 2 || seenKinds[1] != "future_kind" {
		t.Errorf("expected to see future_kind preserved, got %v", seenKinds)
	}
}

func TestScanMissingFileIsEmpty(t *testing.T) {
	s := newTempState(t)
	stats, err := Scan(s, func(*Event, []byte) {})
	if err != nil {
		t.Fatalf("expected nil error on missing file, got %v", err)
	}
	if stats.Parsed != 0 {
		t.Errorf("expected 0 events, got %d", stats.Parsed)
	}
}

func TestPayloadOptInGate(t *testing.T) {
	s := newTempState(t)
	// Default OFF — WritePayloadBlob returns empty without touching disk.
	ref := WritePayloadBlob(s, "id1", []byte("body"))
	if ref != "" {
		t.Errorf("expected empty ref when capture_payloads off, got %q", ref)
	}
	// Opt in
	s.CapturePayloads = true
	ref = WritePayloadBlob(s, "id2", []byte("body"))
	if ref == "" {
		t.Fatal("expected non-empty ref when capture_payloads on")
	}
	if _, err := os.Stat(filepath.Join(s.Dir, ref)); err != nil {
		t.Errorf("expected blob file to exist at %s, err: %v", ref, err)
	}
}
