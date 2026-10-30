package container_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/container"
	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/usecase/search"
	"github.com/jiva-studio/numen/libs/notes/domain"
	_ "modernc.org/sqlite"

	"github.com/jiva-studio/numen/libs/core/port/embeddingport"
)

// makeIndex is an index in a folder of its own.
func makeIndex(t *testing.T) *container.Index {
	t.Helper()
	db, err := container.Config{IndexPath: filepath.Join(t.TempDir(), "index.db")}.OpenIndex(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, db)
	return db
}

// unit is a model of the name given that makes a vector of eight dimensions
// for each text.
type unit struct{ name string }

func (u unit) GetModel() embeddingport.EmbeddingModel {
	return embeddingport.EmbeddingModel{Name: u.name, Dimensions: 8, MaxTokens: 256, Pooling: "mean"}
}

func (unit) Embed(_ context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i := range texts {
		out[i] = []float32{1, 0, 0, 0, 0, 0, 0, 0}
	}
	return out, nil
}

func (unit) Close() error { return nil }

// missingVault is a vault whose folder is not there.
func missingVault(t *testing.T) domain.Vault {
	t.Helper()
	return domain.Vault{ID: "gone", Path: filepath.Join(t.TempDir(), "absent")}
}

func TestAScanConsumesItsTurnsFromTheIndexAndHonoursARebuild(t *testing.T) {
	db := makeIndex(t)

	scan := container.Config{ShouldRebuildIndex: true}.Scan(db)
	if scan.Walks != db.Walks() {
		t.Error("the scan takes its turn from somebody other than the index it writes into")
	}
	if !scan.ShouldRebuildIndex {
		t.Error("a rebuild was asked for and the scan skips files the index remembers")
	}
	if (container.Config{}).Scan(db).ShouldRebuildIndex {
		t.Error("a rebuild was not asked for and the scan reads every file")
	}
}

func TestTheSplitOfASourceIsGivenWhatItReadsWith(t *testing.T) {
	db := makeIndex(t)
	v := testsupport.NewVault(t, map[string]string{"a.md": "# A\n"})
	target := container.Config{Embedding: container.DefaultEmbedding(), ServiceDir: ".numen", ShouldRebuildIndex: true}

	trimmed, err := target.IndexTexts(db.Sources(), db.SourcesKnown(), v)
	if err != nil {
		t.Fatal(err)
	}
	if trimmed.ErrorHandler == nil {
		t.Error("the error handler was not set for a source whose text could not be read")
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
	v := testsupport.NewVault(t, map[string]string{"a.md": "# A\n"})
	model := unit{"first"}

	made, err := container.Config{Embedding: container.DefaultEmbedding(), ServiceDir: ".numen"}.Embed(db, model, v)
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
	target := container.Config{Embedding: container.DefaultEmbedding(), ServiceDir: ".numen"}
	v := missingVault(t)

	if _, err := target.IndexTexts(db.Sources(), db.SourcesKnown(), v); err == nil {
		t.Error("the sources of a folder that is not there were cut")
	}
	if _, err := target.Embed(db, nil, v); err == nil {
		t.Error("the vectors of a folder that is not there were made")
	}
	if _, err := target.ReadWholeVault(t.Context(), db, nil, v); err == nil {
		t.Error("a folder that is not there was read")
	}
}

// owingTo is how many small chunks of the vault the model has no vector for.
func owingTo(t *testing.T, db *container.Index, v domain.Vault, model embeddingport.Embedder) int {
	t.Helper()
	owing, _, err := db.VectorQueries().GetUnembeddedChunks(t.Context(), v.ID, model.GetModel(), "", 1000)
	if err != nil {
		t.Fatal(err)
	}
	return len(owing)
}

// owingAfterAModelSwitch is how many chunks the first model owes a vector for
// once its vault has been made searchable by a second model and then by the
// first again.
func owingAfterAModelSwitch(t *testing.T, shouldRebuild bool) int {
	t.Helper()
	db := makeIndex(t)
	v := testsupport.NewVault(t, map[string]string{"a.md": "# A\n\nsome words to be cut into a chunk\n"})
	target := container.Config{Embedding: container.DefaultEmbedding(), ServiceDir: ".numen"}
	first, second := unit{"first"}, unit{"second"}

	whole, err := target.ReadWholeVault(t.Context(), db, first, v)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := whole.Execute(t.Context(), v); err != nil {
		t.Fatal(err)
	}
	if owingTo(t, db, v, first) != 0 {
		t.Fatal("the first model left chunks without a vector")
	}
	if owingTo(t, db, v, second) == 0 {
		t.Fatal("the vault holds no chunk to be embedded")
	}

	target.ShouldRebuildIndex = shouldRebuild
	if _, err := target.ReadWholeVault(t.Context(), db, second, v); err != nil {
		t.Fatal(err)
	}
	target.ShouldRebuildIndex = false
	if _, err := target.ReadWholeVault(t.Context(), db, first, v); err != nil {
		t.Fatal(err)
	}
	return owingTo(t, db, v, first)
}

// Building the index again is where the vectors of a model nobody asks for any
// more go; every other run keeps them, so a model set back is a model whose
// vectors are all still here.
func TestOnlyARebuildForgetsTheVectorsOfAnotherModel(t *testing.T) {
	if got := owingAfterAModelSwitch(t, false); got != 0 {
		t.Errorf("a run that was not a rebuild forgot the vectors of the first model: %d chunks owe one", got)
	}
	if got := owingAfterAModelSwitch(t, true); got == 0 {
		t.Error("a rebuild kept the vectors of a model unrequested for")
	}
}

// makeSwitchedIndex is an index whose vault was made searchable by one model,
// and the vault, ready to be made searchable by another.
func makeSwitchedIndex(t *testing.T) (*container.Index, domain.Vault) {
	t.Helper()
	db := makeIndex(t)
	v := testsupport.NewVault(t, map[string]string{"a.md": "# A\n\nsome words to be cut into a chunk\n"})
	whole, err := container.Config{Embedding: container.DefaultEmbedding(), ServiceDir: ".numen"}.ReadWholeVault(t.Context(), db, unit{"first"}, v)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := whole.Execute(t.Context(), v); err != nil {
		t.Fatal(err)
	}
	return db, v
}

// A rebuild that cannot forget the vectors of the other model has not rebuilt,
// and says so.
func TestARebuildThatCannotForgetTheOtherVectorsIsRefused(t *testing.T) {
	db, v := makeSwitchedIndex(t)
	raw, err := sql.Open("sqlite", db.Path())
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, raw)
	if _, err := raw.ExecContext(t.Context(),
		`CREATE TRIGGER keep_vectors BEFORE DELETE ON vectors BEGIN SELECT RAISE(ABORT, 'kept'); END`); err != nil {
		t.Fatal(err)
	}

	target := container.Config{Embedding: container.DefaultEmbedding(), ServiceDir: ".numen", ShouldRebuildIndex: true}
	if _, err := target.ReadWholeVault(t.Context(), db, unit{"second"}, v); err == nil || !strings.Contains(err.Error(), "kept") {
		t.Errorf("got %v", err)
	}
}

func TestASearchOverTheIndexWithNoModelIsAnsweredByTheWords(t *testing.T) {
	cfg, vault := makeScannedVault(t)

	found, err := cfg.NewSearch(read(t, cfg), nil, nil).Execute(t.Context(), vault, "disorder", search.Parameters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || found[0].Source != "Entropy.md" {
		t.Errorf("the words answered with %+v", found)
	}
}
