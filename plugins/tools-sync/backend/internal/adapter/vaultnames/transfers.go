package vaultnames

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

type transfers struct {
	inner ports.Transfers
	names *table
}

// Transfers returns what the host does with the bytes of the files of the vault, over the paths of the notes.
func (e *Edge) Transfers(id domain.VaultID, files ports.Files, inner ports.Transfers) ports.Transfers {
	return &transfers{inner: inner, names: e.getTable(id, files)}
}

func (t *transfers) ReadBytes(ctx context.Context, path string, start, length int64) (host.ByteRange, error) {
	part, err := run(ctx, t.names, path, func(name string) (host.ByteRange, error) { return t.inner.ReadBytes(ctx, name, start, length) })
	part.Stat = getStat(part.Stat)
	return part, err
}

func (t *transfers) MoveFile(ctx context.Context, from, to string, baseline *domain.FileStat) (host.MoveFileResult, error) {
	return run(ctx, t.names, from, func(source string) (host.MoveFileResult, error) {
		return create(ctx, t.names, to, func(target string) (host.MoveFileResult, error) {
			return t.inner.MoveFile(ctx, source, target, baseline)
		})
	})
}

func (t *transfers) RemoveFile(ctx context.Context, path string, baseline *domain.FileStat) (host.RemoveFileResult, error) {
	return run(ctx, t.names, path, func(name string) (host.RemoveFileResult, error) { return t.inner.RemoveFile(ctx, name, baseline) })
}

func (t *transfers) PlaceFile(ctx context.Context, name, path, expectedHash string, baseline *domain.FileStat) (domain.FileStat, error) {
	stat, err := create(ctx, t.names, path, func(target string) (domain.FileStat, error) {
		return t.inner.PlaceFile(ctx, name, target, expectedHash, baseline)
	})
	return getStat(stat), err
}
