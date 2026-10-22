package vault_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/embedding"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"

	"github.com/jiva-studio/numen/modules/libs/core/port/embeddingport"
	"github.com/jiva-studio/numen/modules/libs/core/port/vaultport"
)

// A vault made searchable in a terminal and one made searchable in a window are
// the same vault: the three passes, in the one order.
func TestAVaultIsMadeSearchableByThreePasses(t *testing.T) {
	t.Parallel()
	v, readers := vaultAt(t, testsupport.VaultDir(t))
	db := openIndex(t)

	made, err := newWholeVaultRead(readers, db).Execute(t.Context(), v)
	if err != nil {
		t.Fatal(err)
	}
	if made.Notes.Notes == 0 {
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
	t.Parallel()
	v, readers := vaultAt(t, testsupport.VaultDir(t))
	db := openIndex(t)

	if err := db.FitVectors(t.Context(), width, pointing{}.GetModel().Recipe()); err != nil {
		t.Fatal(err)
	}

	var order []string
	notes := scanner(readers, db)
	notes.OnProgress = func(vaults.ScanResult) { order = append(order, "notes") }
	books := source.NewExtract(readers, db.Sources(), db.SourcesKnown())
	books.OnProgress = func(source.ExtractResult) { order = append(order, "books") }
	vectors := embedding.NewEmbed(readers, db.VectorQueries(), db.Vectors())
	vectors.Embedder = pointing{}
	vectors.OnProgress = func(embedding.EmbedResult) { order = append(order, "vectors") }
	making := vaults.NewReadWholeVault(notes, books, vectors)

	if _, err := making.Execute(t.Context(), v); err != nil {
		t.Fatal(err)
	}
	observed := map[string]int{}
	for i, what := range order {
		if _, target := observed[what]; !target {
			observed[what] = i
		}
	}
	for _, what := range []string{"notes", "books", "vectors"} {
		if _, targetVal := observed[what]; !targetVal {
			t.Fatalf("the %s pass did not run: %v", what, order)
		}
	}
	if observed["notes"] > observed["books"] || observed["books"] > observed["vectors"] {
		t.Errorf("the passes ran in the order %v", order)
	}
}

// Reading the notes is what the whole pass relies on.
func TestNotesThatCannotBeReadStopTheRest(t *testing.T) {
	t.Parallel()
	v, readers := vaultAt(t, testsupport.VaultDir(t))
	db := openIndex(t)

	notes := scanner(refusing{}, db)
	counted := &counting{VaultReaders: readers}
	books := source.NewExtract(counted, db.Sources(), db.SourcesKnown())
	vectors := embedding.NewEmbed(readers, db.VectorQueries(), db.Vectors())
	making := vaults.NewReadWholeVault(notes, books, vectors)

	_, err := making.Execute(t.Context(), v)
	if err == nil || !strings.Contains(err.Error(), "not here") {
		t.Fatalf("got %v", err)
	}
	if counted.opened != 0 {
		t.Error("the books were read after the notes could not be")
	}
}

// Two passes that failed are two things wrong with the vault, and the caller is
// told both.
func TestBooksAndVectorsThatBothFailAreBothReported(t *testing.T) {
	t.Parallel()
	v, readers := vaultAt(t, testsupport.VaultDir(t))
	db := openIndex(t)

	if err := db.FitVectors(t.Context(), width, pointing{}.GetModel().Recipe()); err != nil {
		t.Fatal(err)
	}

	notes := scanner(readers, db)
	books := source.NewExtract(refusing{}, db.Sources(), db.SourcesKnown())
	vectors := embedding.NewEmbed(readers, db.VectorQueries(), db.Vectors())
	vectors.Embedder = unwilling{}
	making := vaults.NewReadWholeVault(notes, books, vectors)

	_, err := making.Execute(t.Context(), v)
	if err == nil {
		t.Fatal("a vault whose books and vectors both failed came back clean")
	}
	if !strings.Contains(err.Error(), "not here") {
		t.Errorf("what stopped the books is not in %v", err)
	}
	if !strings.Contains(err.Error(), "not answering") {
		t.Errorf("what stopped the vectors is not in %v", err)
	}
}

// A run given a time limit ends on a different error from one that was
// cancelled, and both are the run being over: the pass that met the limit is
// what the caller is told, and nothing is asked of the context afterwards.
func TestBooksStoppedByATimeLimitAreWhatIsReported(t *testing.T) {
	t.Parallel()
	v, readers := vaultAt(t, testsupport.VaultDir(t))
	db := openIndex(t)

	if err := db.FitVectors(t.Context(), width, pointing{}.GetModel().Recipe()); err != nil {
		t.Fatal(err)
	}

	limitCtx := limit(t.Context())
	notes := scanner(readers, db)
	books := source.NewExtract(timedOut{limit: limitCtx}, db.Sources(), db.SourcesKnown())
	counted := &counting{VaultReaders: readers}
	vectors := embedding.NewEmbed(counted, db.VectorQueries(), db.Vectors())
	vectors.Embedder = unwilling{}
	making := vaults.NewReadWholeVault(notes, books, vectors)

	_, err := making.Execute(limitCtx, v)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got %v", err)
	}
	if !strings.Contains(err.Error(), "books") {
		t.Errorf("the pass that met the limit is not named in %v", err)
	}
	if strings.Contains(err.Error(), "embedding") {
		t.Errorf("the vectors were tried on a context that had ended: %v", err)
	}
	if counted.opened != 0 {
		t.Error("the vectors were tried on a context that had ended")
	}
}

// width is what the vector index is built at.
const width = 1024

// pointing answers every text with one direction.
type pointing struct{}

func (pointing) GetModel() embeddingport.EmbeddingModel {
	return embeddingport.EmbeddingModel{Name: "test", Dimensions: width, MaxTokens: 256, Pooling: "mean"}
}

func (pointing) Embed(_ context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i := range out {
		out[i] = make([]float32, width)
		out[i][0] = 1
	}
	return out, nil
}

func (pointing) Close() error { return nil }

// unwilling is a model that answers nothing, which is what a service that is
// not there looks like from here.
type unwilling struct{}

func (unwilling) GetModel() embeddingport.EmbeddingModel {
	return embeddingport.EmbeddingModel{Name: "test", Dimensions: width, MaxTokens: 256, Pooling: "mean"}
}

func (unwilling) Embed(context.Context, []string) ([][]float32, error) {
	return nil, errors.New("the model is not answering")
}

func (unwilling) Close() error { return nil }

// limited is a run under a time limit, reaching it where the test says. What it
// ends with is the error a deadline gives, which is not the error a cancelled
// run gives.
type limited struct {
	context.Context
	overChan chan struct{}
}

func limit(ctx context.Context) *limited {
	return &limited{Context: ctx, overChan: make(chan struct{})}
}

// reachLimit is the limit being met.
func (l *limited) reachLimit() { close(l.overChan) }

func (l *limited) Done() <-chan struct{} { return l.overChan }

func (l *limited) Err() error {
	select {
	case <-l.overChan:
		return context.DeadlineExceeded
	default:
		return l.Context.Err()
	}
}

// timedOut is a set of readers that reach the run's limit and stop there.
type timedOut struct{ limit *limited }

func (t timedOut) Open(domain.Vault) (vaultport.VaultReader, error) {
	t.limit.reachLimit()
	return nil, t.limit.Err()
}

// refusing opens no vault.
type refusing struct{}

func (refusing) Open(domain.Vault) (vaultport.VaultReader, error) {
	return nil, errors.New("this vault is not here")
}

// counting says how many times a vault was opened for reading.
type counting struct {
	vaultport.VaultReaders
	opened int
}

func (c *counting) Open(v domain.Vault) (vaultport.VaultReader, error) {
	c.opened++
	return c.VaultReaders.Open(v)
}

func newWholeVaultRead(readers vaultport.VaultReaders, db *container.Index) vaults.ReadWholeVault {
	return vaults.NewReadWholeVault(
		scanner(readers, db),
		source.NewExtract(readers, db.Sources(), db.SourcesKnown()),
		embedding.NewEmbed(readers, db.VectorQueries(), db.Vectors()),
	)
}
