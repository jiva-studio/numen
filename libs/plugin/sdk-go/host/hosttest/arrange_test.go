package hosttest_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
)

func TestRenameNoteWritesTheTitleKeyAndMovesTheFile(t *testing.T) {
	v := hosttest.New("Physics")
	v.File("notes/Old.md", "---\ntitle: Old\n---\nbody\n")

	got, err := v.RenameNote(t.Context(), "notes/Old.md", "New")
	if err != nil {
		t.Fatal(err)
	}

	if got.Path != "notes/New.md" || got.Title != "New" || got.IsFilenamed {
		t.Errorf("got %+v", got)
	}
	if got.Move == nil || got.Move.From != "notes/Old.md" || got.Move.To != "notes/New.md" || !got.Move.IsMoved {
		t.Errorf("got move %+v", got.Move)
	}
	content, isThere := v.Content("notes/New.md")
	if !isThere || !strings.Contains(content, "title: New") {
		t.Errorf("got %q, %v", content, isThere)
	}
	if _, isThere := v.Content("notes/Old.md"); isThere {
		t.Error("the old file is still there")
	}
}

func TestRenameNoteOfANoteWithoutTitleKeyMovesTheFileOnly(t *testing.T) {
	v := hosttest.New("Physics")
	v.File("Old.md", "body\n")

	got, err := v.RenameNote(t.Context(), "Old.md", "New")
	if err != nil {
		t.Fatal(err)
	}

	if !got.IsFilenamed || got.Path != "New.md" {
		t.Errorf("got %+v", got)
	}
	if content, _ := v.Content("New.md"); content != "body\n" {
		t.Errorf("got %q", content)
	}
}

func TestRenameNoteOntoAnotherFileIsAlreadyExists(t *testing.T) {
	v := hosttest.New("Physics")
	v.File("Old.md", "body\n")
	v.File("New.md", "other\n")

	if _, err := v.RenameNote(t.Context(), "Old.md", "New"); !errors.Is(err, host.ErrAlreadyExists) {
		t.Errorf("got %v", err)
	}
}

func TestMoveNotesReportsEachPathInOrder(t *testing.T) {
	v := hosttest.New("Physics")
	v.File("a.md", "a")
	v.File("b.md", "b")
	v.File("archive/b.md", "taken")

	got, err := v.MoveNotes(t.Context(), []string{"a.md", "gone.md", "b.md"}, "archive")
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 3 {
		t.Fatalf("got %d moves, want 3", len(got))
	}
	if got[0].Err != nil || got[0].Move.To != "archive/a.md" || !got[0].Move.IsMoved {
		t.Errorf("got %+v", got[0])
	}
	if !errors.Is(got[1].Err, host.ErrNotExist) || got[1].Move.From != "gone.md" || got[1].Move.IsMoved {
		t.Errorf("got %+v", got[1])
	}
	if !errors.Is(got[2].Err, host.ErrAlreadyExists) || got[2].Move.From != "b.md" {
		t.Errorf("got %+v", got[2])
	}
	if content, _ := v.Content("archive/a.md"); content != "a" {
		t.Errorf("got %q", content)
	}
}

func TestRemoveNotesMovesTheNoteToTheTrashAndReportsDanglingLinks(t *testing.T) {
	v := hosttest.New("Physics")
	v.File("Entropy.md", "heat")
	v.File("Notes.md", "see [[Entropy]]")
	v.File("folder/inner.md", "x")

	got, err := v.RemoveNotes(t.Context(), []string{"Entropy.md", "folder", "gone.md"})
	if err != nil {
		t.Fatal(err)
	}

	if got[0].Err != nil || got[0].TrashPath != ".trash/Entropy.md" || !slices.Equal(got[0].BrokenLinks, []string{"Notes.md"}) {
		t.Errorf("got %+v", got[0])
	}
	if !errors.Is(got[1].Err, host.ErrNotANote) {
		t.Errorf("got %+v", got[1])
	}
	if !errors.Is(got[2].Err, host.ErrNotExist) {
		t.Errorf("got %+v", got[2])
	}
	if content, _ := v.Content(".trash/Entropy.md"); content != "heat" {
		t.Errorf("got %q in the trash", content)
	}
	if _, isThere := v.Content("Entropy.md"); isThere {
		t.Error("the removed note is still there")
	}
}

func TestGetProblemsRunsDanglingOnlyWhenNamed(t *testing.T) {
	v := hosttest.New("Physics")
	v.File("Notes.md", "see [[Nowhere]]")

	byDefault, err := v.GetProblems(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	named, err := v.GetProblems(t.Context(), []string{"dangling"})
	if err != nil {
		t.Fatal(err)
	}

	if !slices.Equal(byDefault.Checks, []string{"parse", "frontmatter", "ambiguous"}) || len(byDefault.Found) != 0 {
		t.Errorf("got %+v", byDefault)
	}
	if len(named.Found) != 1 || named.Found[0].Path != "Notes.md" || named.Found[0].Target.Value != "Nowhere" {
		t.Errorf("got %+v", named)
	}
}

func TestGetProblemsFindsAnAmbiguousLink(t *testing.T) {
	v := hosttest.New("Physics")
	v.File("a/Entropy.md", "one")
	v.File("b/Entropy.md", "two")
	v.File("Notes.md", "see [[Entropy]]")

	got, err := v.GetProblems(t.Context(), []string{"ambiguous"})
	if err != nil {
		t.Fatal(err)
	}

	if len(got.Found) != 1 || len(got.Found[0].Candidates) != 2 {
		t.Errorf("got %+v", got)
	}
}

func TestGetProblemsOfAnUnknownCheckIsAnError(t *testing.T) {
	v := hosttest.New("Physics")

	if _, err := v.GetProblems(t.Context(), []string{"spelling"}); err == nil {
		t.Error("got no error")
	}
}
