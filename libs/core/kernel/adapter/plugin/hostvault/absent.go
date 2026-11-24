package hostvault

import (
	"context"
	"io"

	"github.com/jiva-studio/numen/libs/shared/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint/vault"
	vaultdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
)

// Absent lists, fills and evicts the files a vault shows and does not hold. A host that holds every
// file has none.
type Absent struct {
	file.Absent
	StoreReaders func(v domain.Vault, pluginID string) (vault.StoreReader, error)
}

// absentFiles is Absent over the vault of a call.
type absentFiles struct {
	Absent
	session sessionAccess
}

func (w Writer) newAbsent() vault.AbsentFiles {
	if w.Absent == nil {
		return nil
	}
	return absentFiles{Absent: *w.Absent, session: w.session}
}

func (a absentFiles) PutAbsentFiles(ctx context.Context, pluginID string, files []vaultdomain.AbsentFile) error {
	v, err := a.session.open()
	if err != nil {
		return err
	}
	for i := range files {
		files[i].Provider = pluginID
	}
	return a.Absent.Put.Execute(ctx, v, files)
}

func (a absentFiles) ListAbsentFiles(ctx context.Context) ([]vaultdomain.AbsentFile, error) {
	v, err := a.session.open()
	if err != nil {
		return nil, err
	}
	return a.Absent.List.Execute(ctx, v)
}

func (a absentFiles) FillFile(ctx context.Context, pluginID, path, storeName string) error {
	v, err := a.session.open()
	if err != nil {
		return err
	}
	store, err := a.StoreReaders(v, pluginID)
	if err != nil {
		return err
	}
	err = a.Absent.NewFill(storeFiles{store}).Execute(ctx, v, path, storeName)
	a.session.ReportWrites(v.ID, err)
	return err
}

func (a absentFiles) EvictFile(ctx context.Context, pluginID, path, expectedHash string) error {
	v, err := a.session.open()
	if err != nil {
		return err
	}
	err = a.Absent.Evict.Execute(ctx, v, path, expectedHash, pluginID)
	a.session.ReportWrites(v.ID, err)
	return err
}

// storeFiles opens the files of the plugin's store folder.
type storeFiles struct{ vault.StoreReader }

func (s storeFiles) Open(ctx context.Context, name string) (io.ReadCloser, error) {
	opened, _, err := s.StoreReader.Open(ctx, name)
	return opened, err
}

// NewAbsent returns the absent files over the use cases, or nil where there are none.
func NewAbsent(
	cases *file.Absent, storeReaders func(v domain.Vault, pluginID string) (vault.StoreReader, error),
) *Absent {
	if cases == nil {
		return nil
	}
	return &Absent{Absent: *cases, StoreReaders: storeReaders}
}
