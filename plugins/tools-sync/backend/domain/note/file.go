package note

import "time"

// FileSnapshot is a file as the last look at the vault found it.
type FileSnapshot struct {
	Hash       string `json:"hash"`
	Size       int64  `json:"size"`
	ModifiedAt int64  `json:"mtime"`
	SourceID   int64  `json:"sourceId"`
	// SeenAt is when a scan first found the file with this hash and time, in nanoseconds since the Unix epoch.
	SeenAt int64 `json:"seenAt,omitempty"`
}

// IsSettled reports whether the file was left alone for the time at now: its time is that old, or, when its time is in
// the future, a scan found it that long ago.
func (s FileSnapshot) IsSettled(now time.Time, settleTime time.Duration) bool {
	age := now.Sub(time.Unix(0, s.ModifiedAt))
	if age < 0 {
		if s.SeenAt == 0 {
			return false
		}
		age = now.Sub(time.Unix(0, s.SeenAt))
	}
	return age >= settleTime
}
