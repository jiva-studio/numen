package container_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// entropy is the whole of the note the vault below holds.
const entropy = "Entropy is the measure of disorder.\n"

func makeScannedVault(t *testing.T) (container.Config, domain.Vault) {
	t.Helper()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Entropy.md"), []byte(entropy), 0o644); err != nil {
		t.Fatal(err)
	}
	vault := domain.Vault{ID: "01ENTROPY", Name: "physics", Path: dir}
	cfg := container.NewConfig(container.Config{
		Paths: container.Paths{IndexPath: filepath.Join(t.TempDir(), "index.db")},
	})

	db, err := container.OpenIndex(cfg, t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer testsupport.Close(t, db)

	if err := db.Stores.Vaults.Register(t.Context(), vault.ID); err != nil {
		t.Fatal(err)
	}
	err = db.Source.Sources.SaveExtraction(t.Context(), vault.ID, domain.SourceChunks{
		Source: domain.Source{
			FileStat:    domain.FileStat{Path: "Entropy.md", Kind: domain.KindNote, Size: int64(len(entropy)), ModTime: time.Unix(0, 1)},
			ContentHash: "hash-entropy",
			Recipe:      "markdown",
		},
		Chunks: []domain.Chunk{{
			Start: 0, Length: len(entropy), Text: entropy,
			Small: []domain.Chunk{{Start: 0, Length: len(entropy), Text: entropy}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return cfg, vault
}

func read(t *testing.T, cfg container.Config) *vault.Index {
	t.Helper()
	db, err := container.OpenIndex(cfg, t.Context())
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, db)
	return db
}

// An installation with no model searches by words, and that is a whole answer.
func TestSearchOver_WithoutModelUsesWords(t *testing.T) {
	cfg, vault := makeScannedVault(t)

	found, err := cfg.NewSearchOver(search.Index{Passages: read(t, cfg).Chunks.Passages}, nil, nil).
		Execute(t.Context(), vault, "disorder", search.Parameters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 {
		t.Fatalf("the words answered with %d passages", len(found))
	}
	if found[0].Source != "Entropy.md" {
		t.Errorf("the passage came from %q", found[0].Source)
	}
	if found[0].Text != entropy {
		t.Errorf("the passage reads %q", found[0].Text)
	}
}

// A reopened index still lists the vault's sources.
func TestReopenedIndex_KeepsSources(t *testing.T) {
	cfg, vault := makeScannedVault(t)

	target, err := read(t, cfg).Source.SourcesUnder.GetSourcesUnder(t.Context(), vault.ID, "Entropy.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(target) != 1 || target[0].Path != "Entropy.md" {
		t.Fatalf("the vault holds %+v", target)
	}
	if target[0].Kind != domain.KindNote {
		t.Errorf("the note came back as %q", target[0].Kind)
	}
}

// A machine where nothing has scanned reads as a vault holding nothing.
func TestAnIndexThatWasNeverBuiltAnswersEmpty(t *testing.T) {
	pos := filepath.Join(t.TempDir(), "index.db")
	cfg := container.NewConfig(container.Config{
		Paths: container.Paths{IndexPath: pos},
	})
	vault := domain.Vault{ID: "01ENTROPY", Name: "physics", Path: t.TempDir()}

	db := read(t, cfg)
	found, err := cfg.NewSearchOver(search.Index{Passages: db.Chunks.Passages}, nil, nil).
		Execute(t.Context(), vault, "disorder", search.Parameters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 0 {
		t.Errorf("the search answered with %d passages", len(found))
	}
	if target, err := db.Source.SourcesUnder.GetSourcesUnder(t.Context(), vault.ID, ""); err != nil || len(target) != 0 {
		t.Errorf("the vault holds %d sources: %v", len(target), err)
	}
}
