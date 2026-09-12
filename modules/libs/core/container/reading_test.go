package container_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/search"
)

// entropy is the whole of the note the vault below holds.
const entropy = "Entropy is the measure of disorder.\n"

// makeScannedVault is a vault holding one note, and a configuration whose index
// has been built from it. Nothing here is on the machine's own paths.
func makeScannedVault(t *testing.T) (container.Config, domain.Vault) {
	t.Helper()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Entropy.md"), []byte(entropy), 0o644); err != nil {
		t.Fatal(err)
	}
	vault := domain.Vault{ID: "01ENTROPY", Name: "physics", Path: dir}
	cfg := container.Config{IndexPath: filepath.Join(t.TempDir(), "index.db")}

	db, err := cfg.OpenIndex(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.Vaults().Register(t.Context(), vault.ID); err != nil {
		t.Fatal(err)
	}
	err = db.Sources().SaveExtraction(t.Context(), vault.ID, domain.SourceChunks{
		Source: domain.Source{
			Fingerprint: domain.Fingerprint{Path: "Entropy.md", Kind: domain.KindNote, Size: int64(len(entropy)), ModTime: time.Unix(0, 1)},
			Hash:        "hash-entropy",
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

// read is the index of that configuration, opened again beside the one that
// built it.
func read(t *testing.T, cfg container.Config) *container.Index {
	t.Helper()
	db, err := cfg.OpenIndex(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// An installation with no model searches by words, and that is a whole answer.
func TestASearchWithNoModelIsAnsweredByTheWords(t *testing.T) {
	cfg, vault := makeScannedVault(t)

	found, err := cfg.SearchingOver(read(t, cfg).Passages(), nil, nil).
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

// A second opening of the index says what the vault holds.
func TestASecondOpeningOfTheIndexKnowsItsSources(t *testing.T) {
	cfg, vault := makeScannedVault(t)

	held, err := read(t, cfg).SourcesKnown().Under(t.Context(), vault.ID, "Entropy.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(held) != 1 || held[0].Path != "Entropy.md" {
		t.Fatalf("the vault holds %+v", held)
	}
	if held[0].Kind != domain.KindNote {
		t.Errorf("the note came back as %q", held[0].Kind)
	}
}

// A machine where nothing has scanned reads as a vault holding nothing.
func TestAnIndexNobodyHasBuiltAnswersEmpty(t *testing.T) {
	at := filepath.Join(t.TempDir(), "index.db")
	cfg := container.Config{IndexPath: at}
	vault := domain.Vault{ID: "01ENTROPY", Name: "physics", Path: t.TempDir()}

	db := read(t, cfg)
	found, err := cfg.SearchingOver(db.Passages(), nil, nil).
		Execute(t.Context(), vault, "disorder", search.Parameters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 0 {
		t.Errorf("the search answered with %d passages", len(found))
	}
	if held, err := db.SourcesKnown().Under(t.Context(), vault.ID, ""); err != nil || len(held) != 0 {
		t.Errorf("the vault holds %d sources: %v", len(held), err)
	}
}
