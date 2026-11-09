package vaultwriter

import (
	"context"
	"errors"
	"io/fs"
	"os"
	pathpkg "path"
)

// EmptyTrash deletes every file and folder under the vault's trash, deepest
// first, and keeps the trash folder. The paths that could not be deleted are
// returned relative to the vault; an entry that fails does not stop the others.
// A folder holding an undeleted entry stays and is not reported itself; a folder
// that cannot be read is reported as one undeleted path.
func (w *VaultWriter) EmptyTrash(ctx context.Context) (undeleted []string, err error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	vault, err := os.OpenRoot(w.root)
	if err != nil {
		return nil, err
	}
	defer closeAndJoin(&err, vault)
	trash, err := vault.OpenRoot(TrashDir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer closeAndJoin(&err, trash)
	return deleteFolderContent(ctx, trash, ".", TrashDir)
}

func deleteFolderContent(ctx context.Context, root *os.Root, dir, display string) ([]string, error) {
	entries, isReadable := readFolder(root, dir)
	if !isReadable {
		return []string{display}, nil
	}
	var undeleted []string
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return undeleted, err
		}
		left, err := deleteEntry(ctx, root, pathpkg.Join(dir, entry.Name()), pathpkg.Join(display, entry.Name()), entry.IsDir())
		undeleted = append(undeleted, left...)
		if err != nil {
			return undeleted, err
		}
	}
	return undeleted, nil
}

func deleteEntry(ctx context.Context, root *os.Root, name, display string, isDir bool) ([]string, error) {
	if isDir {
		inner, err := deleteFolderContent(ctx, root, name, display)
		if err != nil || len(inner) > 0 {
			return inner, err
		}
	}
	if isGone(root.Remove(name)) {
		return nil, nil
	}
	return []string{display}, nil
}

func readFolder(root *os.Root, dir string) (entries []fs.DirEntry, isReadable bool) {
	folder, err := root.Open(dir)
	if err != nil {
		return nil, false
	}
	entries, err = folder.ReadDir(-1)
	closeAndJoin(&err, folder)
	return entries, err == nil
}

func isGone(removeErr error) bool {
	return removeErr == nil || errors.Is(removeErr, fs.ErrNotExist)
}
