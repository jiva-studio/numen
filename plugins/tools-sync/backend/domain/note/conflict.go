package note

import "time"

// GetConflictCopyPath returns the path of the copy that keeps the other text of a note changed on two computers, in the
// folder of the note, in normal form C. Number 1 has no number in its name.
func GetConflictCopyPath(_ string, _ time.Time, _ int) string { return "" }

// IsCaseClash reports whether two paths differ in letter case or composition only.
func IsCaseClash(_, _ string) bool { return false }
