package telemetry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Log appends one event to events.jsonl. Best-effort: all failures are
// swallowed so telemetry can never break the parent CLI command.
//
// Concurrency: relies on POSIX O_APPEND atomicity for writes smaller than
// PIPE_BUF (4 KB on Linux/macOS). Event payloads are kept well under that
// limit by routing raw bodies to separate blob files.
func Log(s *State, e *Event) {
	if s == nil || s.Disabled || e == nil {
		return
	}
	defer func() { _ = recover() }()

	if e.ID == "" {
		e.ID = NewID()
	}
	if e.Ts == 0 {
		e.Ts = time.Now().UnixMilli()
	}
	if e.Install == "" {
		e.Install = s.InstallID
	}
	if e.V == 0 {
		e.V = SchemaVersion
	}

	// Defensive redaction on the string-y fields that can receive
	// user-controlled content. Most paths here are empty today, but the
	// CHANGELOG / schema-text contract is "Redaction applied at write
	// time" — honour it even on fields that opt-in payload capture (or
	// any future hook) might populate.
	e.Argv = Redact(e.Argv)
	e.ReqPath = Redact(e.ReqPath)
	e.ConfigDelta = Redact(e.ConfigDelta)

	data, err := json.Marshal(e)
	if err != nil {
		return
	}
	if len(data) > 4000 {
		// Hard ceiling — keep each line below PIPE_BUF for append atomicity.
		// Oversized rows almost certainly contain a payload that should be
		// blob-routed instead. Drop rather than risk interleaved writes.
		return
	}
	data = append(data, '\n')

	if err := os.MkdirAll(s.Dir, 0700); err != nil {
		return
	}
	path := filepath.Join(s.Dir, eventsFileName)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(data)
}

// WritePayloadBlob writes a payload body to ~/.thask/payloads/<id>.blob
// with 0600 permissions and returns the relative blob_ref. Caller is
// responsible for confirming s.CapturePayloads is true before invoking.
// Returns empty string on any failure.
func WritePayloadBlob(s *State, id string, body []byte) string {
	if s == nil || s.Disabled || !s.CapturePayloads || len(body) == 0 {
		return ""
	}
	defer func() { _ = recover() }()

	dir := s.PayloadDir()
	if dir == "" {
		return ""
	}
	name := id + ".blob"
	path := filepath.Join(dir, name)
	// Redact tokens / credentials before persisting raw bodies. Body
	// content stays local-only either way, but disk forensics of a
	// stolen ~/.thask should not surface secrets.
	if err := os.WriteFile(path, []byte(RedactHeader(string(body))), 0600); err != nil {
		return ""
	}
	return filepath.Join(payloadDirName, name)
}
