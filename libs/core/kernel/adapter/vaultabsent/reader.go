package vaultabsent

import (
	"context"
	"errors"
	"io"
	"io/fs"
	pathpkg "path"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/shared/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// reader is one vault's inner reader with the entries of its registry. A file whose bytes are held
// wins over the entry of the same path.
type reader struct {
	inner  vaultport.VaultReader
	absent vaultport.AbsentFiles
	vault  domain.Vault
}

func (r reader) WalkFiles(ctx context.Context, fn func(domain.FileStat) error) error {
	held := map[string]bool{}
	err := r.inner.WalkFiles(ctx, func(stat domain.FileStat) error {
		held[stat.Path] = true
		return fn(stat)
	})
	if err != nil {
		return err
	}
	entries, err := r.absent.ListAbsentFiles(ctx, r.vault)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if held[entry.Path] {
			continue
		}
		if err := fn(getStat(entry)); err != nil {
			return err
		}
	}
	return nil
}

func (r reader) List(ctx context.Context, folder string) ([]domain.Entry, error) {
	entries, err := r.inner.List(ctx, folder)
	isMissing := errors.Is(err, fs.ErrNotExist)
	if err != nil && !isMissing {
		return nil, err
	}
	absent, err := r.absent.ListAbsentFiles(ctx, r.vault)
	if err != nil {
		return nil, err
	}
	merged := mergeFolder(entries, absent, folder)
	if isMissing && len(merged) == 0 {
		return nil, fs.ErrNotExist
	}
	return merged, nil
}

func (r reader) Read(ctx context.Context, path string) ([]byte, error) {
	content, err := r.inner.Read(ctx, path)
	if err != nil {
		return nil, r.getReadError(ctx, path, err)
	}
	return content, nil
}

func (r reader) Open(ctx context.Context, path string) (io.ReadSeekCloser, error) {
	file, err := r.inner.Open(ctx, path)
	if err != nil {
		return nil, r.getReadError(ctx, path, err)
	}
	return file, nil
}

func (r reader) StatFile(ctx context.Context, path string) (domain.FileStat, error) {
	stat, err := r.inner.StatFile(ctx, path)
	if !errors.Is(err, fs.ErrNotExist) {
		return stat, err
	}
	entry, isListed, lookupErr := r.absent.GetAbsentFile(ctx, r.vault, path)
	if lookupErr != nil {
		return domain.FileStat{}, lookupErr
	}
	if !isListed {
		return domain.FileStat{}, err
	}
	return getStat(entry), nil
}

// getReadError turns the failure of a read of a path nothing holds into ErrFileAbsent when the
// registry lists the path.
func (r reader) getReadError(ctx context.Context, path string, readErr error) error {
	if !errors.Is(readErr, fs.ErrNotExist) {
		return readErr
	}
	_, isListed, err := r.absent.GetAbsentFile(ctx, r.vault, path)
	if err != nil {
		return err
	}
	if isListed {
		return vault.ErrFileAbsent
	}
	return readErr
}

func getStat(entry vault.AbsentFile) domain.FileStat {
	kind := domain.KindFile
	if strings.EqualFold(pathpkg.Ext(entry.Path), ".md") {
		kind = domain.KindNote
	}
	return domain.FileStat{Path: entry.Path, Kind: kind, Size: entry.Size, ModTime: entry.Mtime, IsAbsent: true}
}

// mergeFolder adds to the entries of a folder the registry files in it and the folders that hold
// registry files deeper down.
func mergeFolder(entries []domain.Entry, absent []vault.AbsentFile, folder string) []domain.Entry {
	folder = strings.Trim(folder, "/")
	have := map[string]bool{}
	for _, entry := range entries {
		have[entry.Path] = true
	}
	for _, file := range absent {
		name, isDeeper, isInside := getChildName(folder, file.Path)
		if !isInside {
			continue
		}
		path := pathpkg.Join(folder, name)
		if have[path] {
			continue
		}
		have[path] = true
		entry := domain.Entry{Path: path, Name: name, IsFolder: isDeeper}
		if !isDeeper {
			stat := getStat(file)
			entry.Kind, entry.ModTime, entry.IsAbsent, entry.Provider = stat.Kind, file.Mtime, true, file.Provider
		}
		entries = append(entries, entry)
	}
	slices.SortStableFunc(entries, compareEntries)
	return entries
}

// getChildName returns the first name of path below folder, and whether it is a folder.
func getChildName(folder, path string) (name string, isFolder, isInside bool) {
	rest := path
	if folder != "" {
		var found bool
		if rest, found = strings.CutPrefix(path, folder+"/"); !found {
			return "", false, false
		}
	}
	name, _, isFolder = strings.Cut(rest, "/")
	return name, isFolder, name != ""
}

func compareEntries(a, b domain.Entry) int {
	if a.IsFolder != b.IsFolder {
		if a.IsFolder {
			return -1
		}
		return 1
	}
	return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
}
