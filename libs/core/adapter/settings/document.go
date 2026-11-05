package settings

import (
	"encoding/json"

	"github.com/jiva-studio/numen/libs/core/adapter/settings/hardware"
)

// File is the file in which the user configures this installation.
type File struct {
	// Version is the format version of the file. Nothing reads it yet.
	Version     int                  `json:"v"`
	Appearance  Appearance           `json:"appearance"`
	Agent       Agent                `json:"agent"`
	Titles      Titles               `json:"naming"`
	Performance hardware.Performance `json:"performance"`
	// Plugins holds the settings of each plugin, keyed by plugin id.
	Plugins map[string]map[string]any `json:"plugins,omitempty"`
	AI      AI                        `json:"ai"`
	// Notices are the warnings that reading the file produced, shown to the
	// user. They are not written to the file.
	Notices []string `json:"-"`
}

// GetVersion returns a pointer to the file format version.
func (s *File) GetVersion() *int { return &s.Version }

// GetAppearance returns a pointer to the appearance section.
func (s *File) GetAppearance() *Appearance { return &s.Appearance }

// AddNotice records one line about the file's contents, shown to the user.
func (s *File) AddNotice(notice string) { s.Notices = append(s.Notices, notice) }

// Sync is whether a note's title and its filename are kept as one name.
func (s File) Sync() bool { return s.Titles.Sync() }

// WriteJSON is every setting as JSON.
func (s File) WriteJSON() (string, error) {
	content, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// DefaultFile is the file of an unconfigured installation.
func DefaultFile() File {
	return File{
		Version:     DefaultVersion,
		Appearance:  DefaultAppearance(),
		Titles:      DefaultTitles(),
		Performance: hardware.DefaultPerformance(),
	}
}
