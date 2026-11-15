// Package profilefiles reads the profiles the person keeps in a folder.
package profilefiles

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/cache"
	"github.com/jiva-studio/numen/plugins/engines-reranking/backend/domain/profile"
)

// Directory is a folder of profile files. An empty path names no folder.
type Directory struct {
	path string
}

// GetDefaultDirectory returns the profiles folder in the data directory of the plugin; it is empty
// when the host grants none.
func GetDefaultDirectory() Directory {
	dir, err := cache.GetDataDir()
	if err != nil {
		return Directory{path: ""}
	}
	return Directory{path: filepath.Join(dir, profile.ProfilesFolder)}
}

// Load returns the profiles that ship with the plugin together with the ones in the folder. A file
// that cannot be read is passed to report and skipped. The error is the one of the shipped profiles, and the
// profiles are returned with it.
func (d Directory) Load(report func(file string, err error)) (*profile.Profiles, error) {
	shipped, err := profile.GetShippedProfiles()
	return shipped.WithUserProfiles(d.path, d.readFiles(report), report), err
}

func (d Directory) readFiles(report func(file string, err error)) []profile.UserFile {
	if d.path == "" {
		return nil
	}
	entries, err := os.ReadDir(d.path)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			report(d.path, err)
		}
		return nil
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.EqualFold(filepath.Ext(entry.Name()), ".json") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	files := make([]profile.UserFile, 0, len(names))
	for _, name := range names {
		path := filepath.Join(d.path, name)
		data, err := os.ReadFile(path)
		if err != nil {
			report(path, err)
			continue
		}
		files = append(files, profile.UserFile{Path: path, Data: data})
	}
	return files
}
