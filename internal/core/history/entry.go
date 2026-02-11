package history

import "time"

// Entry represents a single history entry.
type Entry struct {
	ID           int64
	Method       string
	URL          string
	StatusCode   int
	Duration     time.Duration
	Size         int64
	RequestBody  string
	ResponseBody string
	Headers      string // JSON-encoded request headers
	Timestamp    time.Time
}
