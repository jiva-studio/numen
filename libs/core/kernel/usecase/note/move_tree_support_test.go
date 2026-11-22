package note_test

import (
	"context"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/indexfile"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	vaults "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

type scanIndex struct {
	*index.DB
	walks vaults.Walks
}

func openIndex(t *testing.T) *scanIndex {
	t.Helper()
	db, err := index.Open(t.Context(), indexfile.Path(t))
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, db)
	return &scanIndex{DB: db}
}

func scanner(readers vaultport.VaultReaders, db *scanIndex) vaults.Scan {
	scan := vaults.NewScan(readers, db.Vaults(), db.Notes(), db.NoteQueries(), db.Maintenance(), testsupport.EmptySourceBacklog{})
	scan.Walks = &db.walks
	return scan
}

// countingReaders records the number of files a scan opens.
type countingReaders struct {
	vaultport.VaultReaders
	reads int
}

func (c *countingReaders) Open(v domain.Vault) (vaultport.VaultReader, error) {
	reader, err := c.VaultReaders.Open(v)
	if err != nil {
		return nil, err
	}
	return &countingReader{VaultReader: reader, parent: c}, nil
}

type countingReader struct {
	vaultport.VaultReader
	parent *countingReaders
}

func (c *countingReader) Read(ctx context.Context, path string) ([]byte, error) {
	c.parent.reads++
	return c.VaultReader.Read(ctx, path)
}

func isolateHome(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
}

// unwritable is a set of writers that cannot open any vault for moving.
type unwritable struct {
	err error
}

func (u unwritable) OpenFileMover(domain.Vault) (vaultport.FileMover, error) { return nil, u.err }
