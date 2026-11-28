package vaultfolder

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	vaultdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
)

// timestampTolerance is how much earlier than the clock a file system may stamp what was made now: a
// file system keeps the time of a change in whole seconds or two.
const timestampTolerance = 2 * time.Second

// Folders makes and removes the folders of new vaults on the file system.
type Folders struct{}

// CleanPath returns the shortest path that names the same folder.
func (Folders) CleanPath(path string) string { return filepath.Clean(path) }

// GetParentPath returns the path of the folder that holds the path. A disk root is its own parent.
func (Folders) GetParentPath(path string) string { return filepath.Dir(path) }

// JoinPath returns the path made of the elements.
func (Folders) JoinPath(elements ...string) string { return filepath.Join(elements...) }

// GetFolderState returns whether the path exists, whether it is a folder and whether the folder is empty.
func (Folders) GetFolderState(path string) (vaultdomain.FolderState, error) {
	info, err := os.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return vaultdomain.FolderState{}, nil
	}
	if err != nil {
		return vaultdomain.FolderState{}, fmt.Errorf("stat %s: %w", path, err)
	}
	if !info.IsDir() {
		return vaultdomain.FolderState{Exists: true}, nil
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return vaultdomain.FolderState{}, fmt.Errorf("read the folder %s: %w", path, err)
	}
	return vaultdomain.FolderState{Exists: true, IsFolder: true, IsEmpty: len(entries) == 0}, nil
}

// CreateFolder makes the folder at the path. Its parent exists.
func (Folders) CreateFolder(path string) error {
	return os.Mkdir(path, 0o755)
}

// RemoveCreated removes what the folder holds that was made at or after since. A folder that holds
// anything older keeps all of it.
func (Folders) RemoveCreated(path string, since time.Time) error {
	entries, err := os.ReadDir(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read the folder %s: %w", path, err)
	}
	oldest := since.Add(-timestampTolerance)
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("stat %s: %w", filepath.Join(path, entry.Name()), err)
		}
		if info.ModTime().Before(oldest) {
			return nil
		}
	}
	for _, entry := range entries {
		if err := os.RemoveAll(filepath.Join(path, entry.Name())); err != nil {
			return fmt.Errorf("remove %s: %w", filepath.Join(path, entry.Name()), err)
		}
	}
	return nil
}

// RemoveEmptyFolder removes the folder when it holds nothing. A folder that holds something stays.
func (Folders) RemoveEmptyFolder(path string) error {
	entries, err := os.ReadDir(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read the folder %s: %w", path, err)
	}
	if len(entries) > 0 {
		return nil
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("remove the folder %s: %w", path, err)
	}
	return nil
}
