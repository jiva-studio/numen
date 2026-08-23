package vault_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/container"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/source"
	usecase "github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/vault"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/testsupport"
)

// A vault made searchable in a terminal and one made searchable in a window are
// the same vault: the three passes, in the one order.
func TestAVaultIsMadeSearchableByThreePasses(t *testing.T) {
	v, readers := vaultAt(t, testsupport.VaultDir(t))
	db := openIndex(t)

	made, err := searchable(readers, db).Execute(t.Context(), v)
	if err != nil {
		t.Fatal(err)
	}
	if made.Notes.Seen == 0 {
		t.Error("no note was read")
	}
	if made.Books.Seen == 0 {
		t.Error("no book was seen")
	}
	// With no model nothing is embedded, and that is a whole pass: a vault is
	// searched by its words.
	if made.Vectors.Embedded != 0 {
		t.Errorf("%d vectors were made with no model", made.Vectors.Embedded)
	}
}

// The notes come first, and a vault whose notes could not be read is not a
// vault whose books are read next.
func TestNotesReadBeforeBooksAndBooksBeforeVectors(t *testing.T) {
	v, readers := vaultAt(t, testsupport.VaultDir(t))
	db := openIndex(t)

	if err := db.FitVectors(t.Context(), width); err != nil {
		t.Fatal(err)
	}

	var order []string
	making := searchable(readers, db)
	making.Vectors.Embedder = pointing{}
	making.Notes.OnProgress = func(usecase.ScanResult) { order = append(order, "notes") }
	making.Books.OnProgress = func(source.ExtractResult) { order = append(order, "books") }
	making.Vectors.OnProgress = func(source.EmbedResult) { order = append(order, "vectors") }

	if _, err := making.Execute(t.Context(), v); err != nil {
		t.Fatal(err)
	}
	seen := map[string]int{}
	for i, what := range order {
		if _, held := seen[what]; !held {
			seen[what] = i
		}
	}
	for _, what := range []string{"notes", "books", "vectors"} {
		if _, held := seen[what]; !held {
			t.Fatalf("the %s pass did not run: %v", what, order)
		}
	}
	if seen["notes"] > seen["books"] || seen["books"] > seen["vectors"] {
		t.Errorf("the passes ran in the order %v", order)
	}
}

// Reading the notes is what the whole pass stands on.
func TestNotesThatCannotBeReadStopTheRest(t *testing.T) {
	v, readers := vaultAt(t, testsupport.VaultDir(t))
	db := openIndex(t)

	making := searchable(readers, db)
	making.Notes.Readers = refusing{}
	counted := &counting{VaultReaders: readers}
	making.Books.Readers = counted

	_, err := making.Execute(t.Context(), v)
	if err == nil || !strings.Contains(err.Error(), "not here") {
		t.Fatalf("got %v", err)
	}
	if counted.opened != 0 {
		t.Error("the books were read after the notes could not be")
	}
}

// width is what the vector index is built at.
const width = 1024

// pointing answers every text with one direction.
type pointing struct{}

func (pointing) Model() port.EmbeddingModel {
	return port.EmbeddingModel{Name: "test", Dimensions: width, MaxTokens: 256, Pooling: "mean"}
}

func (pointing) Embed(_ context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i := range out {
		out[i] = make([]float32, width)
		out[i][0] = 1
	}
	return out, nil
}

// refusing opens no vault.
type refusing struct{}

func (refusing) Open(domain.Vault) (port.VaultReader, error) {
	return nil, errors.New("this vault is not here")
}

// counting says how many times a vault was opened for reading.
type counting struct {
	port.VaultReaders
	opened int
}

func (c *counting) Open(v domain.Vault) (port.VaultReader, error) {
	c.opened++
	return c.VaultReaders.Open(v)
}

func searchable(readers port.VaultReaders, db *container.Index) usecase.Searchable {
	return usecase.Searchable{
		Notes: scanner(readers, db),
		Books: source.Extract{
			Readers: readers,
			Sources: db.Sources(),
			Owing:   db.SourcesKnown(),
		},
		Vectors: source.Embed{
			Readers: readers,
			Chunks:  db.VectorsOwing(),
			Vectors: db.Vectors(),
		},
	}
}
