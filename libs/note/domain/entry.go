package domain

import "time"

// Entry is one file or folder of a vault, as a listing of a folder reports it.
type Entry struct {
	Path     string
	Name     string
	IsFolder bool
	Kind     SourceKind
	// ModTime is when the file or folder was last modified. It is zero where
	// the entry is not read from a disk.
	ModTime time.Time
}
