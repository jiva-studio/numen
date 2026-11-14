package vault_test

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/vaultreading"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultfixture"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/filetext"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	vaults "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vectors"
	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/sourcefake"
	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// notRemoving is a list of notes that takes every note but cannot take any out.
type notRemoving struct {
	noteport.NoteRepository
	err error
}

func (n notRemoving) Remove(context.Context, domain.VaultID, []string) error { return n.err }

// uncuttable is an index of sources that cannot report which are still to be
// cut, and runs a function when queried for what was cut by another recipe.
type uncuttable struct {
	sourceport.SourceBacklog
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
	return u.SourceBacklog.GetUnchunkedSources(ctx, id, kind, limit)
}

func (u uncuttable) GetSourcesWithOtherRecipe(
	ctx context.Context, id domain.VaultID, kind domain.SourceKind, recipes []string, limit int,
) ([]string, error) {
	found, err := u.SourceBacklog.GetSourcesWithOtherRecipe(ctx, id, kind, recipes, limit)
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

// Each pass reports its progress to the caller, and the vectors that were
// created are in the result.
func TestEachPassReportsItsProgressToTheCaller(t *testing.T) {
	isolateHome(t)
	v, readers := vaultAt(t, vaultfixture.VaultDir(t))
	db := openIndex(t)
	if err := db.FitVectors(t.Context(), width, pointing{}.Describe().Recipe()); err != nil {
		t.Fatal(err)
	}
	embed := vectors.NewEmbed(readers, db.Sources(), db.Sources())
	embed.Source = sourcefake.Wrap(pointing{})
	making := indexing.NewReadWholeVault(
		scanner(readers, db),
		filetext.NewIndex(readers, db.Sources(), db.Sources(), db.Sources(), db.Sources(), testsupport.Texts{Readers: readers}),
		embed,
	)
	var notes, books, made atomic.Int32

	result, err := making.Execute(t.Context(), v, indexing.Reporting{
		OnNotesProgress:   func(vaultreading.ScanResult) { notes.Add(1) },
		OnTextsProgress:   func(vaultreading.IndexResult) { books.Add(1) },
		OnVectorsProgress: func(vaultreading.EmbedResult) { made.Add(1) },
	})
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
func TestNotesThatStoppedPartwayAreReportedWithWhatWasCompleted(t *testing.T) {
	isolateHome(t)
	v, readers := vaultAt(t, vaultfixture.VaultDir(t))
	db := openIndex(t)
	stopErr := errors.New("the notes cannot be taken out")
	notes := vaults.NewScan(readers, db.Vaults(), notRemoving{NoteRepository: db.Notes(), err: stopErr}, db.NoteQueries(), db.Maintenance(), nil)
	notes.Walks = &db.Walks
	making := indexing.NewReadWholeVault(
		notes,
		filetext.NewIndex(readers, db.Sources(), db.Sources(), db.Sources(), db.Sources(), testsupport.Texts{Readers: readers}),
		vectors.NewEmbed(readers, db.Sources(), db.Sources()),
	)

	result, err := making.Execute(t.Context(), v, indexing.Reporting{})
	if !errors.Is(err, stopErr) {
		t.Fatalf("got %v", err)
	}
	if result.Notes.Notes == 0 {
		t.Error("the answer holds nothing of what the notes pass had done")
	}
}

// Books that stopped partway are an error carrying what stopped them, and the
// answer holds what the pass had done.
func TestBooksThatStoppedPartwayAreReportedWithWhatWasCompleted(t *testing.T) {
	isolateHome(t)
	v, readers := vaultAt(t, vaultfixture.VaultDir(t))
	db := openIndex(t)
	stopErr := errors.New("the books cannot be cut")
	making := indexing.NewReadWholeVault(
		scanner(readers, db),
		filetext.NewIndex(readers, db.Sources(), db.Sources(), uncuttable{SourceBacklog: db.Sources(), err: stopErr}, db.Sources(), testsupport.Texts{Readers: readers}),
		vectors.NewEmbed(readers, db.Sources(), db.Sources()),
	)

	if _, err := scanner(readers, db).Execute(t.Context(), v); err != nil {
		t.Fatal(err)
	}
	read, err := making.IndexTexts(t.Context(), v, indexing.Reporting{})
	if !errors.Is(err, stopErr) {
		t.Fatalf("got %v", err)
	}
	if read.Seen == 0 {
		t.Error("the answer holds nothing of what the books pass had done")
	}
}

// Vectors that could not be made are an error carrying what stopped them, and
// the answer holds what the pass had done.
func TestVectorsThatStoppedAreReportedWithWhatWasCompleted(t *testing.T) {
	isolateHome(t)
	v, readers := vaultAt(t, vaultfixture.VaultDir(t))
	db := openIndex(t)
	if err := db.FitVectors(t.Context(), width, pointing{}.Describe().Recipe()); err != nil {
		t.Fatal(err)
	}
	stopErr := errors.New("the model has stopped")
	embed := vectors.NewEmbed(readers, db.Sources(), db.Sources())
	embed.Source = sourcefake.Wrap(failing{err: stopErr})
	making := indexing.NewReadWholeVault(
		scanner(readers, db),
		filetext.NewIndex(readers, db.Sources(), db.Sources(), db.Sources(), db.Sources(), testsupport.Texts{Readers: readers}),
		embed,
	)

	result, err := making.Execute(t.Context(), v, indexing.Reporting{})
	if !errors.Is(err, stopErr) {
		t.Fatalf("got %v", err)
	}
	if result.Vectors.Pending == 0 {
		t.Error("the answer holds nothing of what the vectors pass had done")
	}
}

// A run that ended while the books were being cut reports what had completed.
func TestReadWholeVault_ReportsCompletedTextsOnBookError(t *testing.T) {
	isolateHome(t)
	v, readers := vaultAt(t, vaultfixture.VaultDir(t))
	db := openIndex(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	stopErr := errors.New("the books cannot be cut")
	uncutter := uncuttable{SourceBacklog: db.Sources(), err: stopErr, onUnchunked: cancel}
	counted := &counting{VaultReaders: readers}
	embed := vectors.NewEmbed(counted, db.Sources(), db.Sources())
	embed.Source = sourcefake.Wrap(pointing{})
	making := indexing.NewReadWholeVault(
		scanner(readers, db),
		filetext.NewIndex(readers, db.Sources(), db.Sources(), uncutter, db.Sources(), testsupport.Texts{Readers: readers}),
		embed,
	)

	result, err := making.Execute(ctx, v, indexing.Reporting{})
	if !errors.Is(err, stopErr) {
		t.Fatalf("got %v", err)
	}
	if result.Texts.Seen == 0 {
		t.Error("the answer holds nothing of what the books pass had done")
	}
	if counted.opened != 0 {
		t.Error("the vectors were tried on a context that had ended")
	}
}

// Books read whole are followed by the vectors, regardless of the context
// state afterwards.
func TestBooksReadWholeAreFollowedByTheVectors(t *testing.T) {
	isolateHome(t)
	v, readers := vaultAt(t, vaultfixture.VaultDir(t))
	db := openIndex(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	uncutter := uncuttable{SourceBacklog: db.Sources(), onRecipe: cancel}
	books := filetext.NewIndex(readers, db.Sources(), db.Sources(), uncutter, db.Sources(), testsupport.Texts{Readers: readers})
	counted := &counting{VaultReaders: readers}
	embed := vectors.NewEmbed(counted, db.Sources(), db.Sources())
	embed.Source = sourcefake.Wrap(pointing{})
	making := indexing.NewReadWholeVault(scanner(readers, db), books, embed)

	_, executeErr := making.Execute(ctx, v, indexing.Reporting{})
	t.Logf("executing the whole read after its context ended gave %v", executeErr)

	if counted.opened == 0 {
		t.Error("the vectors were not tried after the books were read whole")
	}
}

// unreadable is a set of readers that open no vault, with the error they were
// given.
type unreadable struct{ err error }

func (u unreadable) Open(domain.Vault) (vaultport.VaultReader, error) { return nil, u.err }

// One source is cut again from its text, and a source that cannot be cut is an
// error naming it and carrying what stopped it.
func TestOneSourceIsSplitAgain(t *testing.T) {
	isolateHome(t)
	v, readers := vaultAt(t, vaultfixture.VaultDir(t))
	db := openIndex(t)
	making := newWholeVaultRead(readers, db)
	if _, err := making.Execute(t.Context(), v, indexing.Reporting{}); err != nil {
		t.Fatal(err)
	}
	if _, err := making.CutOne(t.Context(), v, "library/A Book.epub"); err != nil {
		t.Errorf("a source that can be cut failed: %v", err)
	}

	stopErr := errors.New("the vault cannot be read")
	cutting := indexing.NewReadWholeVault(
		scanner(readers, db),
		filetext.NewIndex(unreadable{err: stopErr}, db.Sources(), db.Sources(), db.Sources(), db.Sources(), testsupport.Texts{Readers: unreadable{err: stopErr}}),
		vectors.NewEmbed(readers, db.Sources(), db.Sources()),
	)
	_, err := cutting.CutOne(t.Context(), v, "library/A Book.epub")
	if !errors.Is(err, stopErr) {
		t.Fatalf("got %v", err)
	}
	if !strings.Contains(err.Error(), "library/A Book.epub") {
		t.Errorf("the source is not named in %v", err)
	}
}
