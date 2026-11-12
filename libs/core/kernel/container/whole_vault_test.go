package container_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchsource"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultfixture"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
	"github.com/jiva-studio/numen/libs/note/domain"
	_ "modernc.org/sqlite"

	"github.com/jiva-studio/numen/libs/core/kernel/port/searchsourceport"
)

func makeIndex(t *testing.T) *container.Index {
	t.Helper()
	db, err := container.OpenIndex(container.NewConfig(container.Config{
		Paths: container.Paths{IndexPath: filepath.Join(t.TempDir(), "index.db")},
	}), t.Context())
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, db)
	return db
}

// unit is a model of the name given that makes a vector of eight dimensions
// for each text.
type unit struct{ name string }

func (u unit) Describe() searchsource.Description {
	return searchsource.Description{Name: u.name, Width: 8, SourceRecipe: "mean@256"}
}

func (unit) Embed(_ context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i := range texts {
		out[i] = []float32{1, 0, 0, 0, 0, 0, 0, 0}
	}
	return out, nil
}

func (unit) Close() error { return nil }

func missingVault(t *testing.T) domain.Vault {
	t.Helper()
	return domain.Vault{ID: "gone", Path: filepath.Join(t.TempDir(), "absent")}
}

func TestAScanConsumesItsTurnsFromTheIndexAndHonorsARebuild(t *testing.T) {
	db := makeIndex(t)

	scan := container.Scan(container.NewConfig(container.Config{ShouldRebuildIndex: true}), db)
	if scan.Walks != &db.Walks {
		t.Error("the scan takes its turn from somebody other than the index it writes into")
	}
	if !scan.ShouldRebuildIndex {
		t.Error("a rebuild was asked for and the scan skips files the index remembers")
	}
	if container.Scan((container.NewConfig(container.Config{})), db).ShouldRebuildIndex {
		t.Error("a rebuild was not asked for and the scan reads every file")
	}
}

func TestTheSplitOfASourceIsGivenWhatItReadsWith(t *testing.T) {
	db := makeIndex(t)
	v := vaultfixture.NewVault(t, map[string]string{"a.md": "# A\n"})
	target := container.NewConfig(container.Config{
		ShouldRebuildIndex: true,
		Paths:              container.Paths{ServiceDir: ".numen"},
	})

	trimmed, err := container.IndexTexts(target, db, v)
	if err != nil {
		t.Fatal(err)
	}
	if !trimmed.ShouldRebuildIndex {
		t.Error("a rebuild was asked for and the cut skips sources the index remembers")
	}
	if trimmed.Sizes != target.CutNotes().Sizes {
		t.Errorf("cut at %+v, and the settings say %+v", trimmed.Sizes, target.CutNotes().Sizes)
	}
}

func TestTheVectorsOfASourceAreMadeByTheModelGiven(t *testing.T) {
	db := makeIndex(t)
	v := vaultfixture.NewVault(t, map[string]string{"a.md": "# A\n"})
	model := unit{"first"}

	made, err := container.Embed(container.NewConfig(container.Config{
		Paths: container.Paths{ServiceDir: ".numen"},
	}), db, model, v)
	if err != nil {
		t.Fatal(err)
	}
	if made.Claim == nil {
		t.Error("the vectors have no place to claim the index in")
	}
	if made.Texts != nil {
		t.Error("a configuration with no text versions reads text through some")
	}
	if made.Embedder != model {
		t.Errorf("the vectors are made by %v, and the model given is %v", made.Embedder, model)
	}
}

func TestAVaultWhoseFolderIsMissingIsNotMadeSearchable(t *testing.T) {
	db := makeIndex(t)
	target := container.NewConfig(container.Config{
		Paths: container.Paths{ServiceDir: ".numen"},
	})
	v := missingVault(t)

	if _, err := container.IndexTexts(target, db, v); err == nil {
		t.Error("the sources of a folder that is not there were cut")
	}
	if _, err := container.Embed(target, db, nil, v); err == nil {
		t.Error("the vectors of a folder that is not there were made")
	}
	if _, err := container.ReadWholeVault(target, t.Context(), db, nil, v); err == nil {
		t.Error("a folder that is not there was read")
	}
}

func unembeddedTo(t *testing.T, db *container.Index, v domain.Vault, model searchsourceport.Embedder) int {
	t.Helper()
	unembedded, _, err := db.Sources().GetUnembeddedChunks(t.Context(), v.ID, model.Describe(), "", 1000)
	if err != nil {
		t.Fatal(err)
	}
	return len(unembedded)
}

func unembeddedAfterAModelSwitch(t *testing.T, shouldRebuild bool) int {
	t.Helper()
	db := makeIndex(t)
	v := vaultfixture.NewVault(t, map[string]string{"a.md": "# A\n\nsome words to be cut into a chunk\n"})
	target := container.NewConfig(container.Config{
		Paths: container.Paths{ServiceDir: ".numen"},
	})
	first, second := unit{"first"}, unit{"second"}

	whole, err := container.ReadWholeVault(target, t.Context(), db, first, v)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := whole.Execute(t.Context(), v, indexing.Reporting{}); err != nil {
		t.Fatal(err)
	}
	if unembeddedTo(t, db, v, first) != 0 {
		t.Fatal("the first model left chunks without a vector")
	}
	if unembeddedTo(t, db, v, second) == 0 {
		t.Fatal("the vault holds no chunk to be embedded")
	}

	target.ShouldRebuildIndex = shouldRebuild
	if _, err := container.ReadWholeVault(target, t.Context(), db, second, v); err != nil {
		t.Fatal(err)
	}
	target.ShouldRebuildIndex = false
	if _, err := container.ReadWholeVault(target, t.Context(), db, first, v); err != nil {
		t.Fatal(err)
	}
	return unembeddedTo(t, db, v, first)
}

// Only an index rebuild deletes the vectors of a model that is not
// selected; every other run keeps them, so switching back to a model finds all
// of its vectors present.
func TestOnlyARebuildForgetsTheVectorsOfAnotherModel(t *testing.T) {
	if got := unembeddedAfterAModelSwitch(t, false); got != 0 {
		t.Errorf("a run that was not a rebuild forgot the vectors of the first model: %d chunks need one", got)
	}
	if got := unembeddedAfterAModelSwitch(t, true); got == 0 {
		t.Error("a rebuild kept the vectors of a model unrequested for")
	}
}

func makeSwitchedIndex(t *testing.T) (*container.Index, domain.Vault) {
	t.Helper()
	db := makeIndex(t)
	v := vaultfixture.NewVault(t, map[string]string{"a.md": "# A\n\nsome words to be cut into a chunk\n"})
	whole, err := container.ReadWholeVault(container.NewConfig(container.Config{
		Paths: container.Paths{ServiceDir: ".numen"},
	}), t.Context(), db, unit{"first"}, v)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := whole.Execute(t.Context(), v, indexing.Reporting{}); err != nil {
		t.Fatal(err)
	}
	return db, v
}

// A rebuild that cannot delete the vectors of the other model fails and
// reports the error.
func TestARebuildThatCannotForgetTheOtherVectorsIsRefused(t *testing.T) {
	db, v := makeSwitchedIndex(t)
	raw, err := sql.Open("sqlite", db.File)
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, raw)
	if _, err := raw.ExecContext(t.Context(),
		`CREATE TRIGGER keep_vectors BEFORE DELETE ON vectors BEGIN SELECT RAISE(ABORT, 'kept'); END`); err != nil {
		t.Fatal(err)
	}

	target := container.NewConfig(container.Config{
		ShouldRebuildIndex: true,
		Paths:              container.Paths{ServiceDir: ".numen"},
	})
	if _, err := container.ReadWholeVault(target, t.Context(), db, unit{"second"}, v); err == nil || !strings.Contains(err.Error(), "kept") {
		t.Errorf("got %v", err)
	}
}

func TestSearch_WithoutModelUsesWords(t *testing.T) {
	cfg, vault := makeScannedVault(t)

	found, err := container.NewSearch(cfg, read(t, cfg), nil, nil).Execute(t.Context(), vault, "disorder", search.Parameters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || found[0].Source != "Entropy.md" {
		t.Errorf("the words answered with %+v", found)
	}
}
