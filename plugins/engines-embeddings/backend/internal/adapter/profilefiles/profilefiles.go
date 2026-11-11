// Package profilefiles reads the model profiles the user keeps in a folder.
package profilefiles

import (
	"path/filepath"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/cache"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/engine"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/internal/adapter/console"
)

// Directory is a folder of user profile files. An empty path names no folder.
type Directory struct {
	path string
}

// NewDirectory returns the folder at path.
func NewDirectory(path string) Directory {
	return Directory{path: path}
}

// GetDefaultDirectory returns the profiles folder in the data directory of the plugin; it is empty
// when the host grants none.
func GetDefaultDirectory() Directory {
	dir, err := cache.GetDataDir()
	if err != nil {
		return Directory{path: ""}
	}
	return Directory{path: filepath.Join(dir, engine.ProfilesFolder)}
}

// Load returns the profiles that ship with the plugin together with the ones in the folder. A file
// that cannot be read is reported and skipped. The error is the one of the shipped profiles, and the
// profiles are returned with it.
func (d Directory) Load() (*engine.Profiles, error) {
	shipped, err := engine.GetShippedProfiles()
	profiles := shipped.WithUserProfiles(d.path, func(file string, err error) {
		console.Report("embeddings: skipped the model profile "+file, err)
	})
	return profiles, err
}
