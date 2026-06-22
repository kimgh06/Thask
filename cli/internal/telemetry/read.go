package telemetry

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
)

// ScanStats summarises a Scan run for downstream commands.
type ScanStats struct {
	Parsed     int
	Skipped    int
	BytesRead  int64
}

// Visitor is the per-line callback for Scan. Implementations should not
// retain raw beyond the call — copy it if needed.
type Visitor func(e *Event, raw []byte)

// Scan walks events.jsonl line by line, decoding each into Event and
// invoking visit. Missing file is treated as an empty stream (not an
// error). Lines that fail to unmarshal increment Skipped instead of
// halting the scan.
//
// Reader buffer is set to 1 MiB max line size to absorb future schema
// expansion while still rejecting pathological input.
func Scan(s *State, visit Visitor) (ScanStats, error) {
	var stats ScanStats
	if s == nil {
		return stats, nil
	}
	f, err := os.Open(s.EventsPath())
	if err != nil {
		if os.IsNotExist(err) {
			return stats, nil
		}
		return stats, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		stats.BytesRead += int64(len(line)) + 1
		if len(line) == 0 {
			continue
		}
		var e Event
		if err := json.Unmarshal(line, &e); err != nil {
			stats.Skipped++
			continue
		}
		// scanner.Bytes() reuses its buffer; copy for caller retention.
		raw := make([]byte, len(line))
		copy(raw, line)
		visit(&e, raw)
		stats.Parsed++
	}
	if err := scanner.Err(); err != nil && err != io.EOF {
		return stats, err
	}
	return stats, nil
}
