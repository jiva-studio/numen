package vault_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
	usecase "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// A vault made searchable in a terminal and one made searchable in a window are
// the same vault: the three passes, in the one order.
func TestAVaultIsMadeSearchableByThreePasses(t *testing.T) {
	t.Parallel()
	v, readers := vaultAt(t, testsupport.VaultDir(t))
	db := openIndex(t)

	made, err := searchable(readers, db).Execute(t.Context(), v)
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

	if err := db.FitVectors(t.Context(), width, pointing{}.Model().Recipe()); err != nil {
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
	t.Parallel()
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

// Two passes that failed are two things wrong with the vault, and the caller is
// told both.
func TestBooksAndVectorsThatBothFailAreBothReported(t *testing.T) {
	t.Parallel()
	v, readers := vaultAt(t, testsupport.VaultDir(t))
	db := openIndex(t)

	if err := db.FitVectors(t.Context(), width, pointing{}.Model().Recipe()); err != nil {
		t.Fatal(err)
	}

	making := searchable(readers, db)
	making.Books.Readers = refusing{}
	making.Vectors.Embedder = unwilling{}

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

	if err := db.FitVectors(t.Context(), width, pointing{}.Model().Recipe()); err != nil {
		t.Fatal(err)
	}

	over := limit(t.Context())
	making := searchable(readers, db)
	making.Books.Readers = timedOut{limit: over}
	counted := &counting{VaultReaders: readers}
	making.Vectors.Readers = counted
	making.Vectors.Embedder = unwilling{}

	_, err := making.Execute(over, v)
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

func (pointing) Close() error { return nil }

// unwilling is a model that answers nothing, which is what a service that is
// not there looks like from here.
type unwilling struct{}

func (unwilling) Model() port.EmbeddingModel {
	return port.EmbeddingModel{Name: "test", Dimensions: width, MaxTokens: 256, Pooling: "mean"}
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
	over chan struct{}
}

func limit(ctx context.Context) *limited {
	return &limited{Context: ctx, over: make(chan struct{})}
}

// reached is the limit being met.
func (l *limited) reached() { close(l.over) }

func (l *limited) Done() <-chan struct{} { return l.over }

func (l *limited) Err() error {
	select {
	case <-l.over:
		return context.DeadlineExceeded
	default:
		return l.Context.Err()
	}
}

// timedOut is a set of readers that reach the run's limit and stop there.
type timedOut struct{ limit *limited }

func (t timedOut) Open(domain.Vault) (port.VaultReader, error) {
	t.limit.reached()
	return nil, t.limit.Err()
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

func searchable(readers port.VaultReaders, db *container.Index) usecase.ReadWholeVault {
	return usecase.ReadWholeVault{
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
