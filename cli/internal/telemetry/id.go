package telemetry

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"time"
)

// NewID returns a 32-char lowercase hex identifier: 8-byte big-endian Unix
// milliseconds followed by 8 random bytes. Sortable lexicographically by
// time of creation. Across processes the random suffix guarantees
// uniqueness (collision probability ~5e-20 per millisecond).
//
// Per Phase 13 verification §A: cross-process monotonic ordering is NOT
// required. Sort consumers should sort by (ts, id) tuple.
func NewID() string {
	return NewIDAt(time.Now())
}

// NewIDAt is NewID with an explicit time, used by tests.
func NewIDAt(t time.Time) string {
	var buf [16]byte
	binary.BigEndian.PutUint64(buf[:8], uint64(t.UnixMilli()))
	if _, err := rand.Read(buf[8:]); err != nil {
		// Entropy unavailable — fall back to xor-derived bytes. Uniqueness
		// guarantees weaken but writes still succeed (non-interference).
		for i := 8; i < 16; i++ {
			buf[i] = buf[i-8] ^ byte(i)
		}
	}
	return hex.EncodeToString(buf[:])
}
