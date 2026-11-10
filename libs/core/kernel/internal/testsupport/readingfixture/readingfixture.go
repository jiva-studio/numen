// Package readingfixture puts together the parts of an installation that read a vault into a
// cache, for the tests of packages the composition root imports and so cannot import it.
package readingfixture

import (
	"context"
	"path/filepath"
	"testing"

	searchwiring "github.com/jiva-studio/numen/libs/core/kernel/container/search"
	vaultwiring "github.com/jiva-studio/numen/libs/core/kernel/container/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/clock"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwatch"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwriter"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/vaultreading"
	"github.com/jiva-studio/numen/libs/core/kernel/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// Reading reads a vault into an index of its own.
type Reading struct {
	vaults vaultwiring.Config
	search searchwiring.Config
	db     *vault.Index
}

// Open opens an index in a folder of the test and returns what reads into it.
func Open(t *testing.T) (Reading, *vault.Index) {
	t.Helper()

	timer := clock.System{}
	search := searchwiring.Config{Chunks: &note.ChunkTokens{}, Clock: timer, VaultReaders: vaultreader.Readers{}}
	vaults := vaultwiring.Config{
		IndexPath:  filepath.Join(t.TempDir(), "index.db"),
		Clock:      timer,
		VaultLocks: vaultwriter.NewLocks(),
		Watches:    vaultwatch.NewHub(),
		CutNotes:   search.CutNotes(),
		SetNoteCut: search.SetNoteCut,
	}
	db, err := vaults.OpenIndex(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, db)
	return Reading{vaults: vaults, search: search, db: db}, db
}

// ReadWholeVault reads the notes, cuts the file texts and makes the vectors of a vault.
func (r Reading) ReadWholeVault(
	ctx context.Context, embedder embeddingport.Embedder, v domain.Vault, rebuild bool,
) (vaultreading.ReadWholeVault, error) {
	request := r.search
	request.ShouldRebuildIndex = rebuild
	return request.ReadWholeVault(ctx, r.db, embedder, v, r.vaults.MakeRefresh(r.db, r.db.Notes()))
}

// GetVaultReaders opens vaults for reading.
func (r Reading) GetVaultReaders() vaultport.VaultReaders { return r.vaults.VaultReaders() }
