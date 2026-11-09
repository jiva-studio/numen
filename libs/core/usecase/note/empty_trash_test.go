package note_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/usecase/note"
	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

type emptyingWriter struct {
	vaultport.VaultWriter
	undeleted []string
	err       error
	calls     *int
}

func (w emptyingWriter) EmptyTrash(context.Context) ([]string, error) {
	*w.calls++
	return w.undeleted, w.err
}

type emptyingWriters struct {
	vaultport.VaultWriters
	writer  emptyingWriter
	openErr error
}

func (w emptyingWriters) Open(domain.Vault) (vaultport.VaultWriter, error) {
	return w.writer, w.openErr
}

func TestEmptyTrashReturnsWhatTheWriterCouldNotDelete(t *testing.T) {
	t.Parallel()
	calls := 0
	writers := emptyingWriters{writer: emptyingWriter{undeleted: []string{".trash/locked/Heat.md"}, calls: &calls}}

	got, err := note.NewEmptyTrash(writers).Execute(t.Context(), domain.Vault{})
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
	writers := emptyingWriters{writer: emptyingWriter{calls: &calls}, openErr: broken}

	if _, err := note.NewEmptyTrash(writers).Execute(t.Context(), domain.Vault{}); !errors.Is(err, broken) {
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
	writers := emptyingWriters{writer: emptyingWriter{err: broken, calls: &calls}}

	if _, err := note.NewEmptyTrash(writers).Execute(t.Context(), domain.Vault{}); !errors.Is(err, broken) {
		t.Fatalf("want the writer error, got %v", err)
	}
}
