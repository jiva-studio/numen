package domain

import "time"

// Entry is one file or folder of a vault, as a listing of a folder reports it.
type Entry struct {
	Path     string
	Name     string
	IsFolder bool
	Kind     SourceKind
	ModTime  time.Time
}
