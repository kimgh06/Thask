package telemetry

import (
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Session holds per-invocation state assembled by Cobra/HTTP/MCP hooks
// and emitted as one invocation event on Finalize.
type Session struct {
	State *State
	Start time.Time

	mu sync.Mutex

	CmdPath  string
	Source   string
	Format   string
	Flags    []string

	HTTPStatus      int
	BackendHostHash string
	BackendVersion  string
	ResponseBytes   int64
	HadHTTP         bool

	ExitCode   int
	ErrorClass string

	// Reusable invocation id so payload/mcp_call events can reference it.
	InvocationID string

	// Set true after Finalize emitted to make duplicate calls idempotent.
	emitted bool
}

// pkg-level singleton — there is only one CLI invocation per process.
var (
	current   *Session
	currentMu sync.Mutex
)

// Begin establishes the per-invocation session. Safe to call multiple
// times (re-entry returns the existing session).
func Begin(state *State, cliVersion, cmdPath, source string) *Session {
	currentMu.Lock()
	defer currentMu.Unlock()
	if current != nil {
		return current
	}
	s := &Session{
		State:        state,
		Start:        time.Now(),
		CmdPath:      cmdPath,
		Source:       source,
		InvocationID: NewID(),
	}
	current = s
	return s
}

// Current returns the in-flight session, or nil if Begin was never called.
func Current() *Session {
	currentMu.Lock()
	defer currentMu.Unlock()
	return current
}

// SetFormat records the user's selected output format (json/table/quiet).
func (s *Session) SetFormat(f string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.Format = f
	s.mu.Unlock()
}

// SetFlags records the named flags the user supplied. Values are not
// captured here — they may reach payload blobs (opt-in) instead.
func (s *Session) SetFlags(names []string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.Flags = append(s.Flags[:0], names...)
	s.mu.Unlock()
}

// RecordHTTP collapses one HTTP round-trip into invocation-level fields.
// Last call wins for commands that issue multiple requests — coarse but
// sufficient for top-level reporting.
func (s *Session) RecordHTTP(rawURL string, status int, backendVersion string, respBytes int64) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.HTTPStatus = status
	s.BackendHostHash = hashHost(rawURL)
	s.BackendVersion = backendVersion
	s.ResponseBytes = respBytes
	s.HadHTTP = true
	s.mu.Unlock()
}

// SetError classifies the final error from rootCmd.Execute() before
// Finalize emits the invocation event.
func (s *Session) SetError(class string, exitCode int) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.ErrorClass = class
	s.ExitCode = exitCode
	s.mu.Unlock()
}

// Finalize writes the invocation event. Safe to call zero or more times —
// subsequent calls are no-ops.
func (s *Session) Finalize() {
	if s == nil || s.State == nil || s.State.Disabled {
		return
	}
	s.mu.Lock()
	if s.emitted {
		s.mu.Unlock()
		return
	}
	s.emitted = true
	dur := time.Since(s.Start).Milliseconds()
	ev := &Event{
		ID:                s.InvocationID,
		Kind:              KindInvocation,
		Cmd:               s.CmdPath,
		Source:            s.Source,
		DurationMs:        dur,
		ExitCode:          s.ExitCode,
		OK:                s.ErrorClass == "",
		ErrorClass:        s.ErrorClass,
		Format:            s.Format,
		Flags:             append([]string(nil), s.Flags...),
		OutputBytesBucket: BucketBytes(s.ResponseBytes),
	}
	if s.HadHTTP {
		ev.HTTPStatus = s.HTTPStatus
		ev.BackendHostHash = s.BackendHostHash
		ev.BackendVersion = s.BackendVersion
	}
	state := s.State
	s.mu.Unlock()
	Log(state, ev)
}

// hashHost computes sha256(host)[:8 bytes hex = 16 chars]. Returns empty
// for unparsable URLs so we never leak fallback raw URL fragments.
func hashHost(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return ""
	}
	h := sha256.Sum256([]byte(strings.ToLower(u.Host)))
	return hex.EncodeToString(h[:8])
}
