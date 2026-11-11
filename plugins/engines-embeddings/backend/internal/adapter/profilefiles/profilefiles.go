// Package profilefiles reads the model profiles the user keeps in a folder.
package profilefiles

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/cache"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/internal/adapter/console"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/model"
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
	return Directory{path: filepath.Join(dir, model.ProfilesFolder)}
}

// Load returns the profiles that ship with the plugin together with the ones in the folder. A file
// that cannot be read is reported and skipped. The error is the one of the shipped profiles, and the
// profiles are returned with it.
func (d Directory) Load() (*model.Profiles, error) {
	shipped, err := model.GetShippedProfiles()
	report := func(file string, err error) {
		console.Report("embeddings: skipped the model profile "+file, err)
	}
	return shipped.WithUserProfiles(d.path, d.readFiles(report), report), err
}

func (d Directory) readFiles(report func(file string, err error)) []model.UserFile {
	// gomutants:disable-next-line BRANCH_IF reason="os.ReadDir on an empty path returns fs.ErrNotExist and returns no files"
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
	files := make([]model.UserFile, 0, len(names))
	for _, name := range names {
		path := filepath.Join(d.path, name)
		data, err := os.ReadFile(path)
		if err != nil {
			report(path, err)
			continue
		}
		files = append(files, model.UserFile{Path: path, Data: data})
	}
	return files
}
