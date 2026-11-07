package service

import (
	"context"
	"errors"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
)

// FileMove is the move of a note's file and the links the host rewrote for it.
type FileMove struct {
	From          string   `json:"from" jsonschema:"previous path of the note"`
	Path          string   `json:"path" jsonschema:"new path of the note"`
	RepairedNotes []string `json:"repairedNotes" jsonschema:"paths of the notes whose links to the old name were rewritten"`
	BrokenLinks   []string `json:"brokenLinks" jsonschema:"paths of the notes whose links to the old name could not be rewritten and are broken"`
}

// NoteRenameInput is the input of the note_rename tool.
type NoteRenameInput struct {
	Path  string `json:"path" jsonschema:"path of the note"`
	Title string `json:"title" jsonschema:"the new title"`
}

// NoteRenameOutput is the output of the note_rename tool.
type NoteRenameOutput struct {
	Path       string    `json:"path" jsonschema:"path of the note after the rename"`
	Title      string    `json:"title" jsonschema:"title of the note after the rename"`
	NameSource string    `json:"nameSource" jsonschema:"where the title is written: frontmatter for the title key, filename for the file name"`
	Move       *FileMove `json:"move,omitempty" jsonschema:"the move of the file; absent when the file kept its path"`
}

// NoteMoveInput is the input of the note_move tool.
type NoteMoveInput struct {
	Paths  []string `json:"paths" jsonschema:"paths of the notes, at most 50"`
	Folder string   `json:"folder" jsonschema:"folder to move the notes to, relative to the vault folder; \"/\" for the vault root"`
}

// NoteMoveItem is the outcome of moving one note.
type NoteMoveItem struct {
	From          string   `json:"from" jsonschema:"path of the note as given"`
	Path          string   `json:"path,omitempty" jsonschema:"path of the note now; absent when it did not move"`
	RepairedNotes []string `json:"repairedNotes,omitempty" jsonschema:"paths of the notes whose links to the note were rewritten"`
	BrokenLinks   []string `json:"brokenLinks,omitempty" jsonschema:"paths of the notes whose links to the note are now broken"`
	Error         string   `json:"error,omitempty" jsonschema:"error message when the note was not moved"`
}

// NoteMoveOutput is the output of the note_move tool.
type NoteMoveOutput struct {
	Results []NoteMoveItem `json:"results" jsonschema:"one entry per path, in the order given"`
}

// NoteRemoveInput is the input of the note_remove tool.
type NoteRemoveInput struct {
	Paths []string `json:"paths" jsonschema:"paths of the notes, at most 50"`
}

// NoteRemoveItem is the outcome of removing one note.
type NoteRemoveItem struct {
	Path        string   `json:"path" jsonschema:"path of the note as given"`
	TrashPath   string   `json:"trashPath,omitempty" jsonschema:"path of the file in the vault's trash, from which it can be restored"`
	BrokenLinks []string `json:"brokenLinks,omitempty" jsonschema:"paths of the notes whose links to the note are now broken"`
	Error       string   `json:"error,omitempty" jsonschema:"error message when the note was not removed"`
}

// NoteRemoveOutput is the output of the note_remove tool.
type NoteRemoveOutput struct {
	Results []NoteRemoveItem `json:"results" jsonschema:"one entry per path, in the order given"`
}

func (t *Tools) addNoteRename(registry *tools.Registry) error {
	return addTool(registry, tools.Spec{
		Step:  tools.Move,
		Name:  "note_rename",
		Title: "Rename Note",
		Description: "Rename a note. The title is written in the title key when the note has one, " +
			"else in the file name, and the file moves to the file name of the title. The host rewrites " +
			"the links that use the old name of the note.",
		Effect:       tools.ReadWrite,
		Servers:      []tools.Server{tools.Full},
		Instructions: []string{noteRenameSentence, linkRewriteSentence},
	}, inVault(t, (*Tools).renameNote))
}

func (t *Tools) addNoteMove(registry *tools.Registry) error {
	return addTool(registry, tools.Spec{
		Step:  tools.Move,
		Name:  "note_move",
		Title: "Move Notes",
		Description: "Move up to 50 notes to a folder, keeping their file names; the folder \"/\" is the vault " +
			"root. Each path gets its own result, and a path that fails does not stop the others.",
		Effect:       tools.ReadWrite,
		Servers:      []tools.Server{tools.Full},
		Instructions: []string{noteMoveSentence, linkRewriteSentence},
	}, inVault(t, (*Tools).moveNotes))
}

func (t *Tools) addNoteRemove(registry *tools.Registry) error {
	return addTool(registry, tools.Spec{
		Step:  tools.Remove,
		Name:  "note_remove",
		Title: "Remove Notes",
		Description: "Move up to 50 notes to the vault's trash, from which they can be restored. A folder is " +
			"rejected. Each path gets its own result with the notes whose links are now broken.",
		Effect:       tools.ReadWrite,
		Servers:      []tools.Server{tools.Full},
		Instructions: []string{noteRemoveSentence},
	}, inVault(t, (*Tools).removeNotes))
}

func (t *Tools) renameNote(ctx context.Context, in NoteRenameInput) (NoteRenameOutput, error) {
	if strings.TrimSpace(in.Path) == "" {
		return NoteRenameOutput{}, errors.New("path is required")
	}
	result, err := t.vault.RenameNote(ctx, in.Path, in.Title)
	if err != nil {
		return NoteRenameOutput{}, err
	}
	out := NoteRenameOutput{Path: result.Path, Title: result.Title, NameSource: "frontmatter", Move: nil}
	if result.IsFilenamed {
		out.NameSource = "filename"
	}
	if result.Move != nil {
		out.Move = &FileMove{
			From:          result.Move.From,
			Path:          result.Move.To,
			RepairedNotes: getPathList(result.Move.Repaired),
			BrokenLinks:   getPathList(result.Move.Dangling),
		}
	}
	return out, nil
}

func (t *Tools) moveNotes(ctx context.Context, in NoteMoveInput) (NoteMoveOutput, error) {
	if err := checkPaths(in.Paths); err != nil {
		return NoteMoveOutput{}, err
	}
	folder := strings.TrimSpace(in.Folder)
	if folder == "" {
		return NoteMoveOutput{}, errors.New(`folder is empty; pass "/" for the vault root`)
	}
	moves, err := t.vault.MoveNotes(ctx, in.Paths, strings.Trim(folder, "/"))
	if err != nil {
		return NoteMoveOutput{}, err
	}
	out := NoteMoveOutput{Results: make([]NoteMoveItem, 0, len(moves))}
	for _, m := range moves {
		out.Results = append(out.Results, newNoteMoveItem(m))
	}
	return out, nil
}

func newNoteMoveItem(m host.NoteMove) NoteMoveItem {
	item := NoteMoveItem{From: m.Move.From, Path: "", RepairedNotes: nil, BrokenLinks: nil, Error: ""}
	if m.Err != nil {
		item.Error = m.Err.Error()
		return item
	}
	if m.Move.IsMoved {
		item.Path = m.Move.To
	}
	item.RepairedNotes, item.BrokenLinks = m.Move.Repaired, m.Move.Dangling
	return item
}

func (t *Tools) removeNotes(ctx context.Context, in NoteRemoveInput) (NoteRemoveOutput, error) {
	if err := checkPaths(in.Paths); err != nil {
		return NoteRemoveOutput{}, err
	}
	removals, err := t.vault.RemoveNotes(ctx, in.Paths)
	if err != nil {
		return NoteRemoveOutput{}, err
	}
	out := NoteRemoveOutput{Results: make([]NoteRemoveItem, 0, len(removals))}
	for _, r := range removals {
		item := NoteRemoveItem{Path: r.Path, TrashPath: r.TrashPath, BrokenLinks: r.BrokenLinks, Error: ""}
		if r.Err != nil {
			item.TrashPath, item.BrokenLinks, item.Error = "", nil, r.Err.Error()
		}
		out.Results = append(out.Results, item)
	}
	return out, nil
}

// checkPaths rejects a batch of no paths.
func checkPaths(paths []string) error {
	if len(paths) == 0 {
		return errors.New("paths is required")
	}
	return nil
}

// getPathList returns the paths, or an empty list for none, so the result always has the field.
func getPathList(paths []string) []string {
	if paths == nil {
		return []string{}
	}
	return paths
}
