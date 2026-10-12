package mcp

import (
	"context"
	"fmt"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/modules/libs/core/port"
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
			outcome.Refused = sayError(err)
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
			"This removes notes: a path naming a folder is refused, and the notes under " +
			"one are removed by naming each of them.",
	}, newHandler(func(ctx context.Context, in noteRemoveInput) (noteRemoveOutput, error) {
		return removeNotes(ctx, core, in)
	}))
}

func removeNotes(ctx context.Context, core Core, in noteRemoveInput) (noteRemoveOutput, error) {
	if len(in.Paths) > maxRefs {
		return noteRemoveOutput{}, fmt.Errorf("remove at most %d notes at a time", maxRefs)
	}
	reader, err := core.Readers.Open(core.getCurrentVault().Vault)
	if err != nil {
		return noteRemoveOutput{}, err
	}
	res := noteRemoveOutput{Removed: make([]RemoveOutcome, 0, len(in.Paths))}
	for _, path := range in.Paths {
		if err := ctx.Err(); err != nil {
			return noteRemoveOutput{}, err
		}
		res.Removed = append(res.Removed, removeNote(ctx, core, reader, path, in.ShouldDestroy))
	}
	return res, nil
}

// removeNote removes the note at path, or deletes it outright when
// shouldDestroy is set. A refusal is reported in the outcome.
func removeNote(
	ctx context.Context, core Core, reader port.VaultReader, path string, shouldDestroy bool,
) RemoveOutcome {
	if isFolder(ctx, reader, path) {
		return RemoveOutcome{
			RemoveResult: note.RemoveResult{Path: path},
			Refused:      "this is a folder, and this removes notes: name the notes to remove",
		}
	}
	v := core.getCurrentVault().Vault
	var removed note.RemoveResult
	var err error
	if shouldDestroy {
		removed, err = core.Notes.Remove.Destroy(ctx, v, path)
	} else {
		removed, err = core.Notes.Remove.Execute(ctx, v, path)
	}
	outcome := RemoveOutcome{RemoveResult: removed}
	if err != nil {
		outcome.RemoveResult = note.RemoveResult{Path: path}
		outcome.Refused = sayError(err)
	}
	return outcome
}

// isFolder reports whether the vault holds a folder at the path. A folder is
// listed; a path holding a file is not.
func isFolder(ctx context.Context, reader port.VaultReader, path string) bool {
	_, err := reader.List(ctx, path)
	return err == nil
}
