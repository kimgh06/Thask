package telemetry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const (
	dirName             = ".thask"
	eventsFileName      = "events.jsonl"
	tombstoneFileName   = "telemetry-tombstone"
	payloadDirName      = "payloads"
	uploadedLedgerName  = "uploaded.jsonl"
	telemetryConfigName = "telemetry.json"
)

// State is read once per CLI invocation and held package-level. Disabled
// short-circuits every Log() call into a no-op, satisfying Phase 13 §F.
type State struct {
	Disabled        bool
	CapturePayloads bool
	Dir             string
	InstallID       string
	FirstRunAt      int64
}

// LoadState reads ~/.thask/telemetry.json and checks tombstone presence.
// Errors (missing files, permission denied, malformed JSON) are silenced
// per the non-interference rule — telemetry must never break the CLI.
func LoadState() *State {
	defer func() { _ = recover() }()
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, dirName)
	s := &State{Dir: dir}
	if _, err := os.Stat(filepath.Join(dir, tombstoneFileName)); err == nil {
		s.Disabled = true
	}
	if b, err := os.ReadFile(filepath.Join(dir, telemetryConfigName)); err == nil {
		var cfg telemetryConfig
		if json.Unmarshal(b, &cfg) == nil {
			s.CapturePayloads = cfg.CapturePayloads
			s.InstallID = cfg.InstallID
			s.FirstRunAt = cfg.FirstRunAt
		}
	}
	return s
}

type telemetryConfig struct {
	InstallID       string `json:"install_id,omitempty"`
	CapturePayloads bool   `json:"capture_payloads,omitempty"`
	FirstRunAt      int64  `json:"first_run_at,omitempty"`
}

// saveConfig writes ~/.thask/telemetry.json with 0600 permissions.
// Best-effort: returns silently on any failure.
func (s *State) saveConfig() {
	defer func() { _ = recover() }()
	cfg := telemetryConfig{
		InstallID:       s.InstallID,
		CapturePayloads: s.CapturePayloads,
		FirstRunAt:      s.FirstRunAt,
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return
	}
	if err := os.MkdirAll(s.Dir, 0700); err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(s.Dir, telemetryConfigName), data, 0600)
}

// EnsureInstall guarantees an install_id and emits a one-time `install`
// event the first time it runs on this machine.
func EnsureInstall(s *State) {
	if s == nil || s.Disabled {
		return
	}
	if s.InstallID != "" {
		return
	}
	s.InstallID = NewID()
	s.FirstRunAt = time.Now().UnixMilli()
	s.saveConfig()
	Log(s, &Event{
		Kind:          KindInstall,
		OS:            runtime.GOOS,
		Arch:          runtime.GOARCH,
		InstallSource: detectInstallSource(),
		FirstRunAt:    s.FirstRunAt,
	})
}

// detectInstallSource makes a best-effort guess at how `thask` was installed.
// Failure modes return "unknown" — never panics.
func detectInstallSource() string {
	defer func() { _ = recover() }()
	exe, err := os.Executable()
	if err != nil {
		return "unknown"
	}
	switch {
	case containsPath(exe, "/homebrew/"), containsPath(exe, "/Cellar/"):
		return "brew"
	case containsPath(exe, "/node_modules/"), containsPath(exe, "/npm/"):
		return "npm"
	case containsPath(exe, "/go/bin/"):
		return "go-install"
	default:
		return "unknown"
	}
}

func containsPath(s, frag string) bool {
	for i := 0; i+len(frag) <= len(s); i++ {
		if s[i:i+len(frag)] == frag {
			return true
		}
	}
	return false
}

// SetCapturePayloads flips the opt-in flag and persists the change. Also
// emits a `config_change` event so the audit trail explains future
// payload-rich entries. Skipped when telemetry is disabled.
func (s *State) SetCapturePayloads(enabled bool) {
	if s == nil || s.Disabled {
		return
	}
	if s.CapturePayloads == enabled {
		return
	}
	s.CapturePayloads = enabled
	s.saveConfig()
	Log(s, &Event{
		Kind:        KindConfigChange,
		ConfigKey:   "capture_payloads",
		ConfigDelta: boolDelta(enabled),
	})
}

func boolDelta(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// Disable writes a final `config_change` event then drops the tombstone.
// Subsequent Log() calls return immediately.
func Disable(s *State) {
	if s == nil || s.Disabled {
		return
	}
	Log(s, &Event{
		Kind:        KindConfigChange,
		ConfigKey:   "telemetry",
		ConfigDelta: "disabled",
	})
	_ = os.MkdirAll(s.Dir, 0700)
	_ = os.WriteFile(filepath.Join(s.Dir, tombstoneFileName), []byte("disabled\n"), 0600)
	s.Disabled = true
}

// Enable removes the tombstone. The next invocation records the
// resumption via an inline config_change marker on its invocation event.
func Enable(s *State) {
	if s == nil {
		return
	}
	_ = os.Remove(filepath.Join(s.Dir, tombstoneFileName))
	s.Disabled = false
}

// PayloadDir returns the per-payload blob directory, ensuring it exists
// with 0700 permissions. Returns empty string on failure.
func (s *State) PayloadDir() string {
	if s == nil {
		return ""
	}
	p := filepath.Join(s.Dir, payloadDirName)
	if err := os.MkdirAll(p, 0700); err != nil {
		return ""
	}
	return p
}

// EventsPath returns the absolute path to events.jsonl.
func (s *State) EventsPath() string {
	if s == nil {
		return ""
	}
	return filepath.Join(s.Dir, eventsFileName)
}

// TombstonePath returns the absolute path to the opt-out marker.
func (s *State) TombstonePath() string {
	if s == nil {
		return ""
	}
	return filepath.Join(s.Dir, tombstoneFileName)
}
