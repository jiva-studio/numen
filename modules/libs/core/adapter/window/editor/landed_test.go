package editor_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// A move and a rename are several pieces of work over one file. The file lands
// first, and what comes after it can still come apart. The client is told where
// the file is either way, because a person shown a failure whose file did move
// is looking for a note at a path nothing holds.

// stuck is an index that will not file a row under the path a file moved to.
type stuck struct{ port.SourceRepository }

func (stuck) MoveSources(context.Context, domain.VaultID, string, string) error {
	return errors.New("the index would not file the row under the new path")
}

func TestAMoveThatCameApartAfterTheFileLandedSaysWhereItWent(t *testing.T) {
	f := quitting(t, nil, map[string]string{
		"Entropy.md": "---\ntitle: Entropy\n---\n\n# Entropy\n",
	})
	scanned(t, f)
	f.opened.API.Files.Move.Sources = stuck{f.opened.API.Files.Move.Sources}

	answer, err := f.client.MoveFile(t.Context(), connect.NewRequest(&v1.MoveFileRequest{
		From: "Entropy.md",
		To:   "notes/Entropy.md",
	}))
	if err != nil {
		t.Fatalf("the file is at its new path and the client was told %v", err)
	}
	if to := answer.Msg.GetMoved().GetTo(); to != "notes/Entropy.md" {
		t.Errorf("the file the vault holds at notes/Entropy.md is answered for as %q", to)
	}
	if held := onDisk(t, f.root, "notes/Entropy.md"); held == "" {
		t.Error("the file the client was told about holds nothing")
	}
}

func TestARenameThatCameApartAfterTheFileLandedSaysWhatTheNoteIsCalled(t *testing.T) {
	f := quitting(t, nil, map[string]string{
		"Entropy.md": "---\ntitle: Entropy\n---\n\n# Entropy\n",
	})
	scanned(t, f)
	f.opened.API.Notes.Rename.Sources = stuck{f.opened.API.Notes.Rename.Sources}

	answer, err := f.client.RenameNote(t.Context(), connect.NewRequest(&v1.RenameNoteRequest{
		Path:  "Entropy.md",
		Title: "Enthalpy",
	}))
	if err != nil {
		t.Fatalf("the note was written and the client was told %v", err)
	}
	if title := answer.Msg.GetTitle(); title != "Enthalpy" {
		t.Errorf("the note is called %q", title)
	}
	if path := answer.Msg.GetPath(); path != "Enthalpy.md" {
		t.Errorf("the note is filed at %q", path)
	}
}
