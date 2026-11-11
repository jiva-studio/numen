package note

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwriter"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	notecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"
)

type staleNoteWriters struct{ vaultwriter.VaultWriters }

func (o staleNoteWriters) OpenNoteWriter(v domain.Vault) (vaultport.NoteWriter, error) {
	writer, err := o.VaultWriters.OpenNoteWriter(v)
	if err != nil {
		return nil, err
	}
	return staleNoteWriter{NoteWriter: writer}, nil
}

type staleNoteWriter struct{ vaultport.NoteWriter }

func (staleNoteWriter) Write(context.Context, string, []byte, domain.FileStat) (domain.FileStat, error) {
	return domain.FileStat{}, edit.ErrStale
}

type openSession struct{ vault domain.Vault }

func (s openSession) GetCurrentVault() domain.Vault { return s.vault }
func (openSession) BeginWrite() bool                { return true }
func (openSession) FinishWrite()                    {}
func (openSession) IsIndexOutdated(error) bool      { return false }
func (openSession) NoteWritten()                    {}

// A note that another writer changed since it was read is a question for the user.
func TestRenamingANoteChangedElsewhereAnswersWithAConflict(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Old.md"), []byte("---\ntitle: Old\n---\n\n# Old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	deps := notecase.MoveDeps{
		Readers: vaultreader.Readers{}, Writers: staleNoteWriters{VaultWriters: vaultwriter.VaultWriters{Locks: vaultwriter.NewLocks()}},
		Now: testsupport.NewClock(),
	}
	rename := notecase.NewRename(deps, notecase.NewMove(deps))
	session := openSession{vault: domain.Vault{ID: "v1", Name: "v1", Path: root}}
	prose := NewProse(session, nil, nil, nil, nil, &rename)

	out, err := prose.RenameNote(t.Context(), connect.NewRequest(&v1.RenameNoteRequest{Path: "Old.md", Title: "New"}))
	if err != nil {
		t.Fatalf("a note changed elsewhere came back as an error: %v", err)
	}
	if code := out.Msg.GetError(); code != v1.ErrorCode_ERROR_CODE_CONFLICT {
		t.Errorf("a note that changed was answered %v", code)
	}
	if _, err := os.Stat(filepath.Join(root, "Old.md")); err != nil {
		t.Errorf("the file moved for a rename that wrote nothing: %v", err)
	}
}
