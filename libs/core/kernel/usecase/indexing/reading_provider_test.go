package indexing_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/fitindex"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/none"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/chunking"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/derivedfile"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaulttest"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/searchsourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// The tests of the worker read vaults into a real index with the use cases that read the notes, cut the
// file texts and embed the chunks. They are put together here, in the external package, because those use
// cases and the index import this package.
func TestWorker(t *testing.T) {
	indexing.RunWorkerTests(t, func(t *testing.T) (indexing.ReadWholeVaultOpener, vaultport.VaultReaders, *indexing.WorkerIndex) {
		t.Helper()
		db, err := index.Open(t.Context(), filepath.Join(t.TempDir(), "index.db"))
		if err != nil {
			t.Fatal(err)
		}
		testsupport.CloseAfter(t, db)
		var walks indexing.Walks
		ledger := indexing.NewLedger()
		read := func(
			ctx context.Context, source searchsourceport.Source, v domain.Vault, rebuild bool,
		) (indexing.ReadWholeVault, error) {
			if err := fitindex.New(db.VectorIndex(), nil, rebuild).Execute(ctx, source); err != nil {
				return indexing.ReadWholeVault{}, err
			}
			return createReadWholeVault(db, &walks, ledger, source, v, rebuild)
		}
		index := &indexing.WorkerIndex{
			Vaults: db.Vaults(), Notes: db.Notes(), FileStats: db.NoteQueries(),
			Maintenance: db.Maintenance(), Progress: db.ChunkVectors(), Vectors: db.VectorIndex(), Closer: db,
		}
		return read, vaultreader.Readers{}, index
	}, vaulttest.NewVault)
}

func createReadWholeVault(
	db *index.DB, walks *indexing.Walks, ledger *indexing.Ledger, source searchsourceport.Source, v domain.Vault, rebuild bool,
) (indexing.ReadWholeVault, error) {
	readers := vaultreader.Readers{}
	scan := indexing.NewScan(readers, db.Vaults(), db.Notes(), db.NoteQueries(), db.Maintenance(), db.Sources())
	scan.Walks = walks
	refresh := indexing.NewRefreshFrom(indexing.RefreshDeps{
		Readers: readers, Vaults: db.Vaults(), Notes: db.Notes(), Queries: db.Sources(), Sources: db.Sources(),
		Now: testsupport.GetMachine(), Ledger: ledger,
	})
	refresh.Force = true
	findUnreadable := func(ctx context.Context, v domain.Vault, paths []string) ([]string, error) {
		result, err := refresh.Execute(ctx, v, paths)
		return result.Unreadable, err
	}
	cutter := chunking.NewCutNotes(0)
	texts := indexing.NewHandlingIndex(indexing.NewIndex(indexing.IndexDeps{
		Readers: readers, Sources: db.Sources(), Records: db.Sources(), Backlog: db.Sources(), Lookup: db.Sources(),
		Sizes: cutter.Sizes, Legibility: cutter.Legibility, ShouldRebuildIndex: rebuild,
	}), nil, file.NewReadNoteAgain(findUnreadable).Execute)
	store, err := derivedfile.OpenDerived(v.Path, vaultconfig.Options{}, "core")
	if err != nil {
		return indexing.ReadWholeVault{}, err
	}
	if source == nil {
		source = none.SearchSource{}
	}
	embed := indexing.NewEmbed(readers, db.Sources(), db.Sources(), store.NewIndexClaim("core/indexing"), nil, source)
	return indexing.NewReadWholeVault(scan.SetRebuild(rebuild), texts, embed), nil
}
