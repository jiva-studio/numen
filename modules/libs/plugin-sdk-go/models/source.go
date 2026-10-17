package models

import (
	"context"
	"os"
	"path/filepath"
)

// Source is where a plugin keeps its model files and whether it may fetch them.
type Source struct {
	// Section is the settings block a person is told to change.
	Section string
	// Dir is the folder the settings name. Empty is the shared models folder.
	Dir string
	// ShouldDownload allows fetching a file that is not in the folder.
	ShouldDownload bool
	// Progress is told how far a download has got, when anything is listening.
	Progress func(what string, done, total int64)
}

// GetDir returns the folder a fetched file is put in: the folder the settings
// name, or the shared models folder.
func (s Source) GetDir() (string, error) {
	if s.Dir != "" {
		return s.Dir, nil
	}
	return GetHome()
}

// Fetch returns the file one address names, downloaded if it is not already
// here. A sum pins the bytes the address must hold; an empty sum fetches the
// file as named.
func (s Source) Fetch(ctx context.Context, address, sum string) (string, error) {
	dir, err := s.GetDir()
	if err != nil {
		return "", err
	}
	return Fetch(ctx, Request{
		Dir:            dir,
		Section:        s.Section,
		Address:        address,
		Sum:            sum,
		ShouldDownload: s.ShouldDownload,
		Progress:       s.Progress,
	})
}

// GetSearchPaths returns where a file may be: in the folder the settings name,
// and in the folder the application was installed into.
func GetSearchPaths(dir string, names ...string) []string {
	var out []string
	for _, name := range names {
		if dir != "" {
			out = append(out, filepath.Join(dir, name))
		}
		if self, err := os.Executable(); err == nil {
			out = append(out, filepath.Join(filepath.Dir(self), name))
			out = append(out, filepath.Join(filepath.Dir(self), "models", name))
		}
	}
	return out
}
