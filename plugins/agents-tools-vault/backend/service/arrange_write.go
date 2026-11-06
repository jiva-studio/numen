package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
)

// MovedFile is the move of a note's file and the links the host wrote again for it.
type MovedFile struct {
	From     string   `json:"from" jsonschema:"path the note had"`
	To       string   `json:"to" jsonschema:"path the note has now"`
	Repaired []string `json:"repaired" jsonschema:"paths of the notes whose links to the old name were written again"`
	Dangling []string `json:"dangling" jsonschema:"paths of the notes whose links to the old name could not be written again and reach nothing"`
}

// NoteRenameInput is the input of the note_rename tool.
type NoteRenameInput struct {
	Path  string `json:"path" jsonschema:"path of the note"`
	Title string `json:"title" jsonschema:"the new title"`
}

// NoteRenameOutput is the output of the note_rename tool.
type NoteRenameOutput struct {
	Path  string     `json:"path" jsonschema:"path of the note after the rename"`
	Title string     `json:"title" jsonschema:"title of the note after the rename"`
	By    string     `json:"by" jsonschema:"where the title is written: frontmatter for the title key, filename for the file name"`
	Moved *MovedFile `json:"moved,omitempty" jsonschema:"the move of the file; absent when the file kept its path"`
}

// NoteMoveInput is the input of the note_move tool.
type NoteMoveInput struct {
	Paths  []string `json:"paths" jsonschema:"paths of the notes, at most 50"`
	Folder string   `json:"folder" jsonschema:"folder to move the notes to, relative to the vault folder; empty for the vault root"`
}

// NoteMoveItem is the outcome of moving one note.
type NoteMoveItem struct {
	From     string   `json:"from" jsonschema:"path of the note as given"`
	To       string   `json:"to,omitempty" jsonschema:"path of the note now; absent when it did not move"`
	Repaired []string `json:"repaired,omitempty" jsonschema:"paths of the notes whose links to the note were written again"`
	Dangling []string `json:"dangling,omitempty" jsonschema:"paths of the notes whose links to the note reach nothing now"`
	Error    string   `json:"error,omitempty" jsonschema:"why the note did not move"`
}

// NoteMoveOutput is the output of the note_move tool.
type NoteMoveOutput struct {
	Moved []NoteMoveItem `json:"moved" jsonschema:"one entry per path, in the order given"`
}

// NoteRemoveInput is the input of the note_remove tool.
type NoteRemoveInput struct {
	Paths []string `json:"paths" jsonschema:"paths of the notes, at most 50"`
}

// NoteRemoveItem is the outcome of removing one note.
type NoteRemoveItem struct {
	Path     string   `json:"path" jsonschema:"path of the note as given"`
	Trashed  string   `json:"trashed,omitempty" jsonschema:"path of the file in the vault's trash, where it can be put back from"`
	Dangling []string `json:"dangling,omitempty" jsonschema:"paths of the notes whose links to the note reach nothing now"`
	Error    string   `json:"error,omitempty" jsonschema:"why the note was not removed"`
}

// NoteRemoveOutput is the output of the note_remove tool.
type NoteRemoveOutput struct {
	Removed []NoteRemoveItem `json:"removed" jsonschema:"one entry per path, in the order given"`
}

func (t *Tools) addNoteRename(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Step:  tools.Move,
		Name:  "note_rename",
		Title: "Rename Note",
		Description: "Give a note another title. The title is written in the title key when the note has one, " +
			"else in the file name, and the file moves to the file name of the title. Links that named the " +
			"note by its old name are written again by the host.",
		Effect:  tools.ReadWrite,
		Servers: []tools.Server{tools.Full},
	}, inVault(t, (*Tools).renameNote))
}

func (t *Tools) addNoteMove(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Step:  tools.Move,
		Name:  "note_move",
		Title: "Move Notes",
		Description: "Move up to 50 notes to a folder, keeping their file names; an empty folder is the vault " +
			"root. Each path gets its own result, and a path that fails does not stop the others.",
		Effect:  tools.ReadWrite,
		Servers: []tools.Server{tools.Full},
	}, inVault(t, (*Tools).moveNotes))
}

func (t *Tools) addNoteRemove(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Step:  tools.Remove,
		Name:  "note_remove",
		Title: "Remove Notes",
		Description: "Move up to 50 notes to the vault's trash, from where they can be put back. A folder is " +
			"refused. Each path gets its own result with the notes whose links reach nothing now.",
		Effect:  tools.ReadWrite,
		Servers: []tools.Server{tools.Full},
	}, inVault(t, (*Tools).removeNotes))
}

func (t *Tools) renameNote(ctx context.Context, in NoteRenameInput) (NoteRenameOutput, error) {
	if strings.TrimSpace(in.Path) == "" {
		return NoteRenameOutput{}, errors.New("path required")
	}
	if strings.TrimSpace(in.Title) == "" {
		return NoteRenameOutput{}, errors.New("title required")
	}
	renamed, err := t.vault.RenameNote(ctx, in.Path, in.Title)
	if err != nil {
		return NoteRenameOutput{}, fmt.Errorf("rename note %s: %w", in.Path, err)
	}
	out := NoteRenameOutput{Path: renamed.Path, Title: renamed.Title, By: "frontmatter", Moved: nil}
	if renamed.IsFilenamed {
		out.By = "filename"
	}
	if renamed.Moved != nil {
		out.Moved = &MovedFile{
			From:     renamed.Moved.From,
			To:       renamed.Moved.To,
			Repaired: getPathList(renamed.Moved.Repaired),
			Dangling: getPathList(renamed.Moved.Dangling),
		}
	}
	return out, nil
}

func (t *Tools) moveNotes(ctx context.Context, in NoteMoveInput) (NoteMoveOutput, error) {
	if err := checkPaths(in.Paths); err != nil {
		return NoteMoveOutput{}, err
	}
	moves, err := t.vault.MoveNotes(ctx, in.Paths, strings.Trim(in.Folder, "/"))
	if err != nil {
		return NoteMoveOutput{}, fmt.Errorf("move notes: %w", err)
	}
	out := NoteMoveOutput{Moved: make([]NoteMoveItem, 0, len(moves))}
	for _, m := range moves {
		out.Moved = append(out.Moved, newNoteMoveItem(m))
	}
	return out, nil
}

func newNoteMoveItem(m host.NoteMove) NoteMoveItem {
	item := NoteMoveItem{From: m.Moved.From, To: "", Repaired: nil, Dangling: nil, Error: ""}
	if m.Err != nil {
		item.Error = m.Err.Error()
		return item
	}
	if m.Moved.IsMoved {
		item.To = m.Moved.To
	}
	item.Repaired, item.Dangling = m.Moved.Repaired, m.Moved.Dangling
	return item
}

func (t *Tools) removeNotes(ctx context.Context, in NoteRemoveInput) (NoteRemoveOutput, error) {
	if err := checkPaths(in.Paths); err != nil {
		return NoteRemoveOutput{}, err
	}
	removals, err := t.vault.RemoveNotes(ctx, in.Paths)
	if err != nil {
		return NoteRemoveOutput{}, fmt.Errorf("remove notes: %w", err)
	}
	out := NoteRemoveOutput{Removed: make([]NoteRemoveItem, 0, len(removals))}
	for _, r := range removals {
		item := NoteRemoveItem{Path: r.Path, Trashed: r.Trashed, Dangling: r.Dangling, Error: ""}
		if r.Err != nil {
			item.Trashed, item.Dangling, item.Error = "", nil, r.Err.Error()
		}
		out.Removed = append(out.Removed, item)
	}
	return out, nil
}

// checkPaths refuses a batch of no paths or of more than maxBatch.
func checkPaths(paths []string) error {
	switch {
	case len(paths) == 0:
		return errors.New("paths required")
	case len(paths) > maxBatch:
		return fmt.Errorf("%d paths given, at most %d are allowed", len(paths), maxBatch)
	}
	return nil
}

// getPathList returns the paths, or an empty list for none, so the result always carries the field.
func getPathList(paths []string) []string {
	if paths == nil {
		return []string{}
	}
	return paths
}
