package dto

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// PaginationMeta is included in paginated v1 list responses.
type PaginationMeta struct {
	Limit      int     `json:"limit"`
	HasMore    bool    `json:"hasMore"`
	NextCursor *string `json:"nextCursor,omitempty"`
}

// PaginatedResponse wraps data with pagination metadata for v1.
type PaginatedResponse struct {
	Data       any            `json:"data"`
	Pagination PaginationMeta `json:"pagination"`
}

// EncodeCursor encodes a (createdAt, id) pair into a base64 cursor string.
func EncodeCursor(createdAt time.Time, id string) string {
	raw := fmt.Sprintf("%s|%s", createdAt.UTC().Format(time.RFC3339Nano), id)
	return base64.URLEncoding.EncodeToString([]byte(raw))
}

// DecodeCursor decodes a base64 cursor into (createdAt, id).
func DecodeCursor(cursor string) (time.Time, string, error) {
	raw, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return time.Time{}, "", fmt.Errorf("invalid cursor encoding")
	}
	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 {
		return time.Time{}, "", fmt.Errorf("invalid cursor format")
	}
	id := parts[1]
	if !uuidRegex.MatchString(id) {
		return time.Time{}, "", fmt.Errorf("invalid cursor id format")
	}
	t, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, "", fmt.Errorf("invalid cursor timestamp")
	}
	return t, id, nil
}

// ClampLimit ensures limit is between 1 and max (default 50, max 100).
func ClampLimit(limit, max int) int {
	if max <= 0 {
		max = 100
	}
	if limit <= 0 {
		return 50
	}
	if limit > max {
		return max
	}
	return limit
}
