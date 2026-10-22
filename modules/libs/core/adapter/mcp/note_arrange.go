package mcp

import (
	"context"
	"fmt"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

type noteRenameInput struct {
	Path  string `json:"path" jsonschema:"the note to rename"`
	Title string `json:"title" jsonschema:"what it is called from now on"`
}

func addNoteRename(server *sdk.Server, core Core) {
	addTool(server, &sdk.Tool{
		Name:  "note_rename",
		Title: "Rename a note",
		Description: "Give a note a different name. " + namingOrder + " Whichever of the " +
			"three names it is brought into line, and the file is renamed with it. A note " +
			"its filename names is moved and not written. The answer says which of them " +
			"named it, and what the file did. Links written by the old name are repaired " +
			"only where they stopped resolving.",
	}, newHandler(func(ctx context.Context, in noteRenameInput) (note.RenameResult, error) {
		return core.Notes.Rename.Execute(ctx, core.getCurrentVault().Vault, in.Path, in.Title)
	}))
}

type noteMoveInput struct {
	Paths  []string `json:"paths" jsonschema:"the notes to move"`
	Folder string   `json:"folder" jsonschema:"where they go, relative to the vault folder; empty is the root"`
}

type noteMoveOutput struct {
	Moved []MoveOutcome `json:"moved"`
}

func addNoteMove(server *sdk.Server, core Core) {
	addTool(server, &sdk.Tool{
		Name:  "note_move",
		Title: "Move a note",
		Description: "File notes under a different folder, keeping their names. Folders " +
			"are for how the files are arranged on disk and change nothing about the " +
			"graph. Links written by name follow the note.",
	}, newHandler(func(ctx context.Context, in noteMoveInput) (noteMoveOutput, error) {
		return moveNotes(ctx, core, in)
	}))
}

func moveNotes(ctx context.Context, core Core, in noteMoveInput) (noteMoveOutput, error) {
	if len(in.Paths) > maxRefs {
		return noteMoveOutput{}, fmt.Errorf("move at most %d notes at a time", maxRefs)
	}
	res := noteMoveOutput{Moved: make([]MoveOutcome, 0, len(in.Paths))}
	for _, path := range in.Paths {
		if err := ctx.Err(); err != nil {
			return noteMoveOutput{}, err
		}
		moved, err := core.Notes.Move.Execute(ctx, core.getCurrentVault().Vault, path, note.GetPathUnder(in.Folder, path))
		outcome := MoveOutcome{MoveResult: moved}
		if err != nil {
			outcome.MoveResult = note.MoveResult{From: path}
			outcome.Error = sayError(err)
		}
		res.Moved = append(res.Moved, outcome)
	}
	return res, nil
}

type noteRemoveInput struct {
	Paths         []string `json:"paths" jsonschema:"the notes to remove"`
	ShouldDestroy bool     `json:"destroy,omitempty" jsonschema:"delete outright instead of moving to the trash; nothing brings these back"`
}

type noteRemoveOutput struct {
	Removed []RemoveOutcome `json:"removed"`
}

func addNoteRemove(server *sdk.Server, core Core) {
	addTool(server, &sdk.Tool{
		Name:  "note_remove",
		Title: "Remove a note",
		Description: "Take notes out of the vault. They go to the vault's trash folder " +
			"and can be put back — unless the call sets `destroy`, which takes the file " +
			"off the disk and leaves nothing to put back. " +
			"Links that pointed at them are left as they are and " +
			"come back under `dangling`: a link is not wrong because its note is gone. " +
			"This removes notes: a path naming a folder fails, and the notes under " +
			"one are removed by naming each of them.",
	}, newHandler(func(ctx context.Context, in noteRemoveInput) (noteRemoveOutput, error) {
		return removeNotes(ctx, core, in)
	}))
}

func removeNotes(ctx context.Context, core Core, in noteRemoveInput) (noteRemoveOutput, error) {
	if len(in.Paths) > maxRefs {
		return noteRemoveOutput{}, fmt.Errorf("remove at most %d notes at a time", maxRefs)
	}
	res := noteRemoveOutput{Removed: make([]RemoveOutcome, 0, len(in.Paths))}
	for _, path := range in.Paths {
		if err := ctx.Err(); err != nil {
			return noteRemoveOutput{}, err
		}
		removed, err := core.Notes.RemoveNote.Execute(ctx, core.getCurrentVault().Vault, path, in.ShouldDestroy)
		outcome := RemoveOutcome{RemoveResult: removed}
		if err != nil {
			outcome.RemoveResult = note.RemoveResult{Path: path}
			outcome.Error = sayError(err)
		}
		res.Removed = append(res.Removed, outcome)
	}
	return res, nil
}
