package settings

import (
	"errors"
	"io/fs"
	"os"
)

// Performance is the settings section for workload and resource profile.
type Performance struct {
	Profile               string `json:"profile"`
	DisableDmabufRenderer bool   `json:"disableDmabufRenderer"`
}

// ReadDisableDmabufRenderer reads performance.disableDmabufRenderer from the settings file at path without
// writing anything. A file that is not there or cannot be read leaves the default.
func ReadDisableDmabufRenderer(path string) bool {
	if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
		return DefaultPerformance().DisableDmabufRenderer
	}
	file, err := (&Store{files: NewFiles()}).ReadUncheckedAt(path)
	if err != nil {
		return DefaultPerformance().DisableDmabufRenderer
	}
	return file.Performance.DisableDmabufRenderer
}
