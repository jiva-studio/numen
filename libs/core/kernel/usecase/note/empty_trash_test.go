package note_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

type trashWriterStub struct {
	undeleted []string
	err       error
	calls     *int
}

func (w trashWriterStub) EmptyTrash(context.Context) ([]string, error) {
	*w.calls++
	return w.undeleted, w.err
}

type trashWriterProviderStub struct {
	writer  trashWriterStub
	openErr error
}

func (w trashWriterProviderStub) OpenTrashEmptier(domain.Vault) (vaultport.TrashEmptier, error) {
	return w.writer, w.openErr
}

func TestEmptyTrashReturnsWhatTheWriterCouldNotDelete(t *testing.T) {
	t.Parallel()
	calls := 0
	writers := trashWriterProviderStub{writer: trashWriterStub{undeleted: []string{".trash/locked/Heat.md"}, calls: &calls}}

	got, err := note.NewEmptyTrash(writers, noTrashedVersions{}, note.OpenGate{}).Execute(t.Context(), domain.Vault{})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Errorf("the trash was emptied %d times", calls)
	}
	if !slices.Equal(got, []string{".trash/locked/Heat.md"}) {
		t.Errorf("undeleted: %v", got)
	}
}

func TestEmptyTrashStopsWhenTheVaultCannotBeOpened(t *testing.T) {
	t.Parallel()
	calls := 0
	broken := errors.New("the vault is gone")
	writers := trashWriterProviderStub{writer: trashWriterStub{calls: &calls}, openErr: broken}

	if _, err := note.NewEmptyTrash(writers, noTrashedVersions{}, note.OpenGate{}).Execute(t.Context(), domain.Vault{}); !errors.Is(err, broken) {
		t.Fatalf("want the open error, got %v", err)
	}
	if calls != 0 {
		t.Error("the trash was emptied in a vault that did not open")
	}
}

func TestEmptyTrashReturnsTheWriterError(t *testing.T) {
	t.Parallel()
	calls := 0
	broken := errors.New("the disk is gone")
	writers := trashWriterProviderStub{writer: trashWriterStub{err: broken, calls: &calls}}

	if _, err := note.NewEmptyTrash(writers, noTrashedVersions{}, note.OpenGate{}).Execute(t.Context(), domain.Vault{}); !errors.Is(err, broken) {
		t.Fatalf("want the writer error, got %v", err)
	}
}

func TestEmptyTrashReturnsTheUndeletedPathsTogetherWithTheWriterError(t *testing.T) {
	t.Parallel()
	calls := 0
	broken := errors.New("the disk is gone")
	writers := trashWriterProviderStub{writer: trashWriterStub{undeleted: []string{".trash/Heat.md"}, err: broken, calls: &calls}}

	got, err := note.NewEmptyTrash(writers, noTrashedVersions{}, note.OpenGate{}).Execute(t.Context(), domain.Vault{})
	if !errors.Is(err, broken) {
		t.Fatalf("want the writer error, got %v", err)
	}
	if !slices.Equal(got, []string{".trash/Heat.md"}) {
		t.Errorf("undeleted: %v", got)
	}
}

// noTrashedVersions is a trash whose files have no versions.
type noTrashedVersions struct{}

func (noTrashedVersions) DeleteUnder(context.Context, domain.Vault, string, []string) error {
	return nil
}
