package vault_test

import (
	"context"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/indexfile"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
)

type scanIndex struct {
	*index.DB
	walks indexing.Walks
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

func scanner(readers vaultport.VaultReaders, db *scanIndex) indexing.Scan {
	scan := indexing.NewScan(readers, db.Vaults(), db.Notes(), db.NoteQueries(), db.Maintenance(), testsupport.EmptySourceBacklog{})
	scan.Walks = &db.walks
	return scan
}

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

func getNotesOnly() []domain.SourceKind { return []domain.SourceKind{domain.KindNote} }
