package vault_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// notRemoving is a list of notes that takes every note but cannot take any out.
type notRemoving struct {
	port.NoteRepository
	err error
}

func (n notRemoving) Remove(context.Context, domain.VaultID, []string) error { return n.err }

// uncuttable is an index of sources that cannot say which are still to be cut,
// and runs a function when asked what was cut by another recipe.
type uncuttable struct {
	port.SourceQueries
	err         error
	onUnchunked func()
	onRecipe    func()
}

func (u uncuttable) GetUnchunkedSources(
	ctx context.Context, id domain.VaultID, kind domain.SourceKind, limit int,
) ([]string, error) {
	if u.onUnchunked != nil {
		u.onUnchunked()
	}
	if u.err != nil {
		return nil, u.err
	}
	return u.SourceQueries.GetUnchunkedSources(ctx, id, kind, limit)
}

func (u uncuttable) ByOtherRecipe(
	ctx context.Context, id domain.VaultID, kind domain.SourceKind, recipes []string, limit int,
) ([]string, error) {
	found, err := u.SourceQueries.ByOtherRecipe(ctx, id, kind, recipes, limit)
	if u.onRecipe != nil {
		u.onRecipe()
	}
	return found, err
}

// failing is a model that answers nothing, with the error it was given.
type failing struct {
	unwilling
	err error
}

func (f failing) Embed(context.Context, []string) ([][]float32, error) { return nil, f.err }

// Each pass tells the caller how far it has got, and the vectors that were made
// are in the answer.
func TestEachPassReportsItsProgressToTheCaller(t *testing.T) {
	isolateHome(t)
	v, readers := vaultAt(t, testsupport.VaultDir(t))
	db := openIndex(t)
	if err := db.FitVectors(t.Context(), width, pointing{}.Model().Recipe()); err != nil {
		t.Fatal(err)
	}
	vectors := source.NewEmbed(readers, db.VectorsOwing(), db.Vectors())
	vectors.Embedder = pointing{}
	making := vaults.NewReadWholeVault(
		scanner(readers, db),
		source.NewExtract(readers, db.Sources(), db.SourcesKnown()),
		vectors,
	)
	var notes, books, made atomic.Int32
	making.OnNotesProgress = func(vaults.ScanResult) { notes.Add(1) }
	making.OnBooksProgress = func(source.ExtractResult) { books.Add(1) }
	making.OnVectorsProgress = func(source.EmbedResult) { made.Add(1) }

	result, err := making.Execute(t.Context(), v)
	if err != nil {
		t.Fatal(err)
	}

	if notes.Load() == 0 {
		t.Error("the caller was not told how far the notes had got")
	}
	if books.Load() == 0 {
		t.Error("the caller was not told how far the books had got")
	}
	if made.Load() == 0 {
		t.Error("the caller was not told how far the vectors had got")
	}
	if result.Vectors.Embedded == 0 {
		t.Error("the answer holds no vectors")
	}
}

// Notes that could not all be read are an error naming the vault, carrying what
// stopped them, and the answer holds what the pass had done.
func TestNotesThatStoppedPartwayAreReportedWithWhatWasDone(t *testing.T) {
	isolateHome(t)
	v, readers := vaultAt(t, testsupport.VaultDir(t))
	db := openIndex(t)
	stopped := errors.New("the notes cannot be taken out")
	notes := vaults.NewScan(readers, db.Vaults(), notRemoving{NoteRepository: db.Notes(), err: stopped}, db.Queries(), db.Maintenance())
	notes.Walks = db.Walks()
	making := vaults.NewReadWholeVault(
		notes,
		source.NewExtract(readers, db.Sources(), db.SourcesKnown()),
		source.NewEmbed(readers, db.VectorsOwing(), db.Vectors()),
	)

	result, err := making.Execute(t.Context(), v)
	if !errors.Is(err, stopped) {
		t.Fatalf("got %v", err)
	}
	if result.Notes.Notes == 0 {
		t.Error("the answer holds nothing of what the notes pass had done")
	}
}

// Books that stopped partway are an error carrying what stopped them, and the
// answer holds what the pass had done.
func TestBooksThatStoppedPartwayAreReportedWithWhatWasDone(t *testing.T) {
	isolateHome(t)
	v, readers := vaultAt(t, testsupport.VaultDir(t))
	db := openIndex(t)
	stopped := errors.New("the books cannot be cut")
	making := vaults.NewReadWholeVault(
		scanner(readers, db),
		source.NewExtract(readers, db.Sources(), uncuttable{SourceQueries: db.SourcesKnown(), err: stopped}),
		source.NewEmbed(readers, db.VectorsOwing(), db.Vectors()),
	)

	read, err := making.ReadBooks(t.Context(), v)
	if !errors.Is(err, stopped) {
		t.Fatalf("got %v", err)
	}
	if read.Seen == 0 {
		t.Error("the answer holds nothing of what the books pass had done")
	}
}

// Vectors that could not be made are an error carrying what stopped them, and
// the answer holds what the pass had done.
func TestVectorsThatStoppedAreReportedWithWhatWasDone(t *testing.T) {
	isolateHome(t)
	v, readers := vaultAt(t, testsupport.VaultDir(t))
	db := openIndex(t)
	if err := db.FitVectors(t.Context(), width, pointing{}.Model().Recipe()); err != nil {
		t.Fatal(err)
	}
	stopped := errors.New("the model has stopped")
	vectors := source.NewEmbed(readers, db.VectorsOwing(), db.Vectors())
	vectors.Embedder = failing{err: stopped}
	making := vaults.NewReadWholeVault(
		scanner(readers, db),
		source.NewExtract(readers, db.Sources(), db.SourcesKnown()),
		vectors,
	)

	result, err := making.Execute(t.Context(), v)
	if !errors.Is(err, stopped) {
		t.Fatalf("got %v", err)
	}
	if result.Vectors.Owing == 0 {
		t.Error("the answer holds nothing of what the vectors pass had done")
	}
}

// A run that ended while the books were being cut is told what the books had
// done.
func TestARunThatEndedInTheBooksIsToldWhatTheBooksHadDone(t *testing.T) {
	isolateHome(t)
	v, readers := vaultAt(t, testsupport.VaultDir(t))
	db := openIndex(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	stopped := errors.New("the books cannot be cut")
	cut := uncuttable{SourceQueries: db.SourcesKnown(), err: stopped, onUnchunked: cancel}
	counted := &counting{VaultReaders: readers}
	vectors := source.NewEmbed(counted, db.VectorsOwing(), db.Vectors())
	vectors.Embedder = pointing{}
	making := vaults.NewReadWholeVault(
		scanner(readers, db),
		source.NewExtract(readers, db.Sources(), cut),
		vectors,
	)

	result, err := making.Execute(ctx, v)
	if !errors.Is(err, stopped) {
		t.Fatalf("got %v", err)
	}
	if result.Books.Seen == 0 {
		t.Error("the answer holds nothing of what the books pass had done")
	}
	if counted.opened != 0 {
		t.Error("the vectors were tried on a context that had ended")
	}
}

// Books read whole are followed by the vectors, whatever the context says
// afterwards.
func TestBooksReadWholeAreFollowedByTheVectors(t *testing.T) {
	isolateHome(t)
	v, readers := vaultAt(t, testsupport.VaultDir(t))
	db := openIndex(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	cut := uncuttable{SourceQueries: db.SourcesKnown(), onRecipe: cancel}
	books := source.NewExtract(readers, db.Sources(), cut)
	books.Kinds = []domain.SourceKind{domain.KindBook}
	counted := &counting{VaultReaders: readers}
	vectors := source.NewEmbed(counted, db.VectorsOwing(), db.Vectors())
	vectors.Embedder = pointing{}
	making := vaults.NewReadWholeVault(scanner(readers, db), books, vectors)

	_, _ = making.Execute(ctx, v)

	if counted.opened == 0 {
		t.Error("the vectors were not tried after the books were read whole")
	}
}
