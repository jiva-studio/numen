// Package arrangewire calls the arrange use case with the messages of the note_rename, note_move,
// note_remove and vault_problem_list tools.
package arrangewire

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/api"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/usecase/arrange"
)

// RenameNote answers note_rename.
func RenameNote(ctx context.Context, u *arrange.UseCase, in api.NoteRenameInput) (api.NoteRenameOutput, error) {
	result, err := u.Rename(ctx, in.Path, in.Title)
	if err != nil {
		return api.NoteRenameOutput{}, err
	}
	out := api.NoteRenameOutput{Path: result.Path, Title: result.Title, NameSource: "frontmatter", Move: nil}
	if result.IsFilenamed {
		out.NameSource = "filename"
	}
	if result.Move != nil {
		out.Move = &api.FileMove{
			From:          result.Move.From,
			Path:          result.Move.To,
			RepairedNotes: getPathList(result.Move.Repaired),
			BrokenLinks:   getPathList(result.Move.Dangling),
		}
	}
	return out, nil
}

// MoveNotes answers note_move.
func MoveNotes(ctx context.Context, u *arrange.UseCase, in api.NoteMoveInput) (api.NoteMoveOutput, error) {
	moves, err := u.Move(ctx, in.Paths, in.Folder)
	if err != nil {
		return api.NoteMoveOutput{}, err
	}
	out := api.NoteMoveOutput{Results: make([]api.NoteMoveItem, 0, len(moves))}
	for _, move := range moves {
		out.Results = append(out.Results, newMoveItem(move))
	}
	return out, nil
}

// RemoveNotes answers note_remove.
func RemoveNotes(ctx context.Context, u *arrange.UseCase, in api.NoteRemoveInput) (api.NoteRemoveOutput, error) {
	removals, err := u.Remove(ctx, in.Paths)
	if err != nil {
		return api.NoteRemoveOutput{}, err
	}
	out := api.NoteRemoveOutput{Results: make([]api.NoteRemoveItem, 0, len(removals))}
	for _, removal := range removals {
		item := api.NoteRemoveItem{Path: removal.Path, TrashPath: removal.TrashPath, BrokenLinks: removal.BrokenLinks, Error: ""}
		if removal.Err != nil {
			item.TrashPath, item.BrokenLinks, item.Error = "", nil, removal.Err.Error()
		}
		out.Results = append(out.Results, item)
	}
	return out, nil
}

// ListProblems answers vault_problem_list.
func ListProblems(ctx context.Context, u *arrange.UseCase, in api.VaultProblemListInput) (api.VaultProblemListOutput, error) {
	problems, err := u.ListProblems(ctx, in.Checks)
	if err != nil {
		return api.VaultProblemListOutput{}, err
	}
	out := api.VaultProblemListOutput{
		Problems: make([]api.VaultProblemItem, 0, len(problems.Found)),
		Checks:   problems.Checks,
	}
	if out.Checks == nil {
		out.Checks = []string{}
	}
	for _, problem := range problems.Found {
		address := ""
		if problem.Target.Value != "" {
			address = problem.Target.GetWritten()
		}
		out.Problems = append(out.Problems, api.VaultProblemItem{
			Path:       problem.Path,
			Check:      string(problem.Kind),
			Detail:     problem.Detail,
			Address:    address,
			Candidates: problem.Candidates,
		})
	}
	return out, nil
}

func newMoveItem(move host.NoteMove) api.NoteMoveItem {
	item := api.NoteMoveItem{From: move.Move.From, Path: "", RepairedNotes: nil, BrokenLinks: nil, Error: ""}
	if move.Err != nil {
		item.Error = move.Err.Error()
		return item
	}
	if move.Move.IsMoved {
		item.Path = move.Move.To
	}
	item.RepairedNotes, item.BrokenLinks = move.Move.Repaired, move.Move.Dangling
	return item
}

func getPathList(paths []string) []string {
	if paths == nil {
		return []string{}
	}
	return paths
}
