package indexing_test

import (
	"path/filepath"
	"testing"

	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/vaultfixture"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// The tests of the worker read vaults into an index through the composition root, which imports
// this package and so can only be imported by its external tests.
func TestWorker(t *testing.T) {
	indexing.RunWorkerTests(t, func(t *testing.T) (indexing.OpenReading, vaultport.VaultReaders, *indexing.WorkerIndex) {
		t.Helper()
		cfg := container.NewConfig(container.Config{
			Paths: container.Paths{IndexPath: filepath.Join(t.TempDir(), "index.db")},
		})
		db, err := container.OpenIndex(cfg, t.Context())
		if err != nil {
			t.Fatal(err)
		}
		testsupport.CloseAfter(t, db)
		read := func(
			ctx context.Context, embedder embeddingport.Embedder, v domain.Vault, rebuild bool,
		) (indexing.ReadWholeVault, error) {
			request := cfg
			request.ShouldRebuildIndex = rebuild
			return container.ReadWholeVault(request, ctx, db, embedder, v)
		}
		index := &indexing.WorkerIndex{
			Vaults: db.Vaults(), Notes: db.Notes(), FileStats: db.NoteQueries(),
			Maintenance: db.Maintenance(), Progress: db.ChunkVectors(), Vectors: db, Closer: db,
		}
		return read, container.VaultReaders(cfg), index
	}, vaultfixture.NewVault)
}
