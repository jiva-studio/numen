package vaultnames

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

type absent struct {
	inner ports.Absent
	names *table
}

// Absent returns the files of the vault that the device lists and does not hold, over the paths of the notes.
func (e *Edge) Absent(id domain.VaultID, files ports.Files, inner ports.Absent) ports.Absent {
	return &absent{inner: inner, names: e.getTable(id, files)}
}

func (a *absent) PutAbsentFiles(ctx context.Context, listed []host.AbsentFile) error {
	return a.inner.PutAbsentFiles(ctx, listed)
}

func (a *absent) ListAbsentFiles(ctx context.Context) ([]host.AbsentFile, error) {
	return a.inner.ListAbsentFiles(ctx)
}

func (a *absent) FillFile(ctx context.Context, path, storeName string) error {
	return a.inner.FillFile(ctx, path, storeName)
}

func (a *absent) EvictFile(ctx context.Context, path, expectedHash string) error {
	_, err := run(ctx, a.names, path, func(name string) (struct{}, error) { return struct{}{}, a.inner.EvictFile(ctx, name, expectedHash) })
	return err
}
