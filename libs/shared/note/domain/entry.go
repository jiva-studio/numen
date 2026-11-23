package domain

import "time"

// Entry is one file or folder of a vault, as a listing of a folder reports it.
type Entry struct {
	Path     string
	Name     string
	IsFolder bool
	Kind     SourceKind
	ModTime  time.Time
	// IsAbsent is set when the vault lists the file and its bytes are not on this device.
	IsAbsent bool
	// Provider is the id of the plugin that can fill an absent file.
	Provider string
}
