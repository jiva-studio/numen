package models

import (
	"context"
	"os"
	"path/filepath"
)

// Folder is where a plugin keeps its model files and whether it may fetch them.
type Folder struct {
	// Section is the settings block a person is told to change.
	Section string
	// Dir is the folder the settings name. Empty is the shared models folder.
	Dir string
	// ShouldDownload allows fetching a file that is not in the folder.
	ShouldDownload bool
	// Progress is told how far a download has got, when anything is listening.
	Progress func(what string, done, total int64)
}

// NewFolder returns a Folder for a settings section.
func NewFolder(section, dir string, shouldDownload bool, progress func(what string, done, total int64)) Folder {
	return Folder{
		Section:        section,
		Dir:            dir,
		ShouldDownload: shouldDownload,
		Progress:       progress,
	}
}

// GetDir returns the folder a fetched file is put in: the folder the settings
// name, or the shared models folder.
func (f Folder) GetDir() (string, error) {
	if f.Dir != "" {
		return f.Dir, nil
	}
	return GetHome()
}

// Fetch returns the file one address names, downloaded if it is not already
// here. A sum pins the bytes the address must hold; an empty sum fetches the
// file as named.
func (f Folder) Fetch(ctx context.Context, address, sum string) (string, error) {
	dir, err := f.GetDir()
	if err != nil {
		return "", err
	}
	return Fetch(ctx, Request{
		Dir:            dir,
		Section:        f.Section,
		Address:        address,
		Sum:            sum,
		ShouldDownload: f.ShouldDownload,
		Progress:       f.Progress,
	})
}

// GetPaths returns every place a file may be: in the folder the settings name,
// and in the folder the application was installed into.
func (f Folder) GetPaths(file string) []string {
	var out []string
	if f.Dir != "" {
		out = append(out, filepath.Join(f.Dir, file))
	}
	if self, err := os.Executable(); err == nil {
		out = append(out, filepath.Join(filepath.Dir(self), file))
		out = append(out, filepath.Join(filepath.Dir(self), "models", file))
	}
	return out
}

// Source returns the places to look for a model file and the way to fetch one.
func (f Folder) Source() Source {
	return Source{GetPaths: f.GetPaths, Fetch: f.Fetch}
}

// Locate finds and fetches each wanted model file, writing its path to its destination.
func (f Folder) Locate(ctx context.Context, pins Pins, wanted []Wanted) error {
	return LocateAll(ctx, f.Source(), pins, wanted)
}

