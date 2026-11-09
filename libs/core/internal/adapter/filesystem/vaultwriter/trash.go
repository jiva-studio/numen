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

// deleteFolderContent empties dir of root. A folder holding an undeleted entry
// stays and is not reported itself.
func deleteFolderContent(ctx context.Context, root *os.Root, dir, display string) ([]string, error) {
	entries, err := readFolder(root, dir)
	if err != nil {
		return []string{display}, nil
	}
	var undeleted []string
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return undeleted, err
		}
		name := pathpkg.Join(dir, entry.Name())
		displayName := pathpkg.Join(display, entry.Name())
		if entry.IsDir() {
			inner, err := deleteFolderContent(ctx, root, name, displayName)
			undeleted = append(undeleted, inner...)
			if err != nil {
				return undeleted, err
			}
			if len(inner) > 0 {
				continue
			}
		}
		if err := root.Remove(name); err != nil && !errors.Is(err, fs.ErrNotExist) {
			undeleted = append(undeleted, displayName)
		}
	}
	return undeleted, nil
}

func readFolder(root *os.Root, dir string) ([]fs.DirEntry, error) {
	folder, err := root.Open(dir)
	if err != nil {
		return nil, err
	}
	defer func() { _ = folder.Close() }()
	return folder.ReadDir(-1)
}
