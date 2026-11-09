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
// A folder holding an undeleted entry stays unreported; an unreadable folder is
// one undeleted path. A trash that is a link is removed and its target kept.
func (w *VaultWriter) EmptyTrash(ctx context.Context) (undeleted []string, err error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	vault, err := os.OpenRoot(w.root)
	if err != nil {
		return nil, err
	}
	defer closeAndJoin(&err, vault)
	info, err := vault.Lstat(TrashDir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if info.Mode()&fs.ModeSymlink != 0 {
		if isGone(vault.Remove(TrashDir)) {
			return nil, nil
		}
		return []string{TrashDir}, nil
	}
	trash, err := vault.OpenRoot(TrashDir)
	if err != nil {
		return nil, err
	}
	defer closeAndJoin(&err, trash)
	if !isSameFolder(info, trash) {
		return []string{TrashDir}, nil
	}
	return deleteFolderContent(ctx, trash, ".", TrashDir)
}

func deleteFolderContent(ctx context.Context, root *os.Root, dir, display string) ([]string, error) {
	entries, err := readFolder(root, dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
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

func readFolder(root *os.Root, dir string) (entries []fs.DirEntry, err error) {
	folder, err := root.Open(dir)
	if err != nil {
		return nil, err
	}
	entries, err = folder.ReadDir(-1)
	closeAndJoin(&err, folder)
	return entries, err
}

// isSameFolder reports whether the opened root is the folder that was checked,
// so a link swapped in between the check and the open is never walked.
func isSameFolder(checked fs.FileInfo, opened *os.Root) bool {
	current, err := opened.Stat(".")
	return err == nil && os.SameFile(checked, current)
}

func isGone(removeErr error) bool {
	return removeErr == nil || errors.Is(removeErr, fs.ErrNotExist)
}
