package vaultnames

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

type files struct {
	inner ports.Files
	names *table
}

// Files returns the files of the vault with the paths of the notes: the paths the feed lists are in normal form C,
// and a path of a call reaches the file by its disk name.
func (e *Edge) Files(id domain.VaultID, inner ports.Files) ports.Files {
	return &files{inner: inner, names: e.getTable(id, inner)}
}

func (f *files) StatFile(ctx context.Context, path string) (domain.FileStat, error) {
	stat, err := run(ctx, f.names, path, func(name string) (domain.FileStat, error) { return f.inner.StatFile(ctx, name) })
	return getStat(stat), err
}

type contentRead struct {
	content []byte
	stat    domain.FileStat
}

func (f *files) Read(ctx context.Context, path string) ([]byte, domain.FileStat, error) {
	read, err := run(ctx, f.names, path, func(name string) (contentRead, error) {
		content, stat, err := f.inner.Read(ctx, name)
		return contentRead{content: content, stat: stat}, err
	})
	return read.content, getStat(read.stat), err
}

func (f *files) Write(ctx context.Context, path string, content []byte, baseline domain.FileStat) (domain.FileStat, error) {
	stat, err := run(ctx, f.names, path, func(name string) (domain.FileStat, error) { return f.inner.Write(ctx, name, content, baseline) })
	return getStat(stat), err
}

func (f *files) Create(ctx context.Context, path string, content []byte) error {
	_, err := create(ctx, f.names, path, func(name string) (struct{}, error) { return struct{}{}, f.inner.Create(ctx, name, content) })
	return err
}

func (f *files) ListFileChanges(ctx context.Context, q host.FileChangeQuery) (host.FileChangePage, error) {
	page, err := f.inner.ListFileChanges(ctx, q)
	if err != nil {
		return page, err
	}
	f.names.observe(page.Changes)
	changes := make([]host.FileChange, len(page.Changes))
	for i, change := range page.Changes {
		change.Path = getNote(change.Path)
		changes[i] = change
	}
	page.Changes = changes
	return page, nil
}
