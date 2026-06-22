package telemetry

// Kind values. New kinds may be added without schema migration —
// readers ignore unknown kinds. Never reuse a kind name with new semantics.
const (
	KindInstall      = "install"
	KindInvocation   = "invocation"
	KindMCPCall      = "mcp_call"
	KindPayload      = "payload"
	KindConfigChange = "config_change"
)

// SchemaVersion is written into the `v` field. Bump when an existing kind's
// field meaning changes (never when adding new fields).
const SchemaVersion = 1

// Source values.
const (
	SourceTerminal = "terminal"
	SourceMCP      = "mcp"
)

// Event is one JSONL line. Field names follow the locked-in vocabulary
// (no abbreviations, no overloaded names like "status"). Adding optional
// fields is safe — readers preserve unknown JSON keys via raw-line capture.
type Event struct {
	// Required on every line.
	ID      string `json:"id"`
	Ts      int64  `json:"ts"`
	Install string `json:"install"`
	Kind    string `json:"kind"`
	V       int    `json:"v"`

	// Cross-kind optional.
	Parent    string `json:"parent,omitempty"`
	LocalOnly bool   `json:"local_only,omitempty"`

	// invocation kind.
	CLIVersion        string   `json:"cli_version,omitempty"`
	Cmd               string   `json:"cmd,omitempty"`
	Source            string   `json:"source,omitempty"`
	DurationMs        int64    `json:"duration_ms,omitempty"`
	ExitCode          int      `json:"exit_code,omitempty"`
	OK                bool     `json:"ok,omitempty"`
	ErrorClass        string   `json:"error_class,omitempty"`
	BackendHostHash   string   `json:"backend_host_hash,omitempty"`
	BackendVersion    string   `json:"backend_version,omitempty"`
	HTTPStatus        int      `json:"http_status,omitempty"`
	Format            string   `json:"format,omitempty"`
	Flags             []string `json:"flags,omitempty"`
	OutputBytesBucket string   `json:"output_bytes_bucket,omitempty"`

	// mcp_call kind.
	ToolName         string `json:"tool_name,omitempty"`
	MCPClient        string `json:"mcp_client,omitempty"`
	AgentModel       string `json:"agent_model,omitempty"`
	AgentSessionHash string `json:"agent_session_hash,omitempty"`
	PayloadInBucket  string `json:"payload_in_bucket,omitempty"`
	PayloadOutBucket string `json:"payload_out_bucket,omitempty"`

	// payload kind (capture-payloads opt-in only).
	BlobKind  string `json:"blob_kind,omitempty"`
	BlobSize  int64  `json:"blob_size,omitempty"`
	BlobRef   string `json:"blob_ref,omitempty"`
	Argv      string `json:"argv,omitempty"`
	ReqMethod string `json:"req_method,omitempty"`
	ReqPath   string `json:"req_path,omitempty"`

	// install kind.
	OS            string `json:"os,omitempty"`
	Arch          string `json:"arch,omitempty"`
	InstallSource string `json:"install_source,omitempty"`
	Shell         string `json:"shell,omitempty"`
	FirstRunAt    int64  `json:"first_run_at,omitempty"`

	// config_change kind.
	ConfigKey   string `json:"config_key,omitempty"`
	ConfigDelta string `json:"config_delta,omitempty"`
}

// BucketBytes maps a byte count to a coarse bucket label used as
// output_bytes_bucket / payload_*_bucket. Buckets are stable strings so
// future-version readers can group them without numeric reparsing.
func BucketBytes(n int64) string {
	switch {
	case n < 1<<10:
		return "<1k"
	case n < 10*(1<<10):
		return "<10k"
	case n < 100*(1<<10):
		return "<100k"
	default:
		return ">=100k"
	}
}
