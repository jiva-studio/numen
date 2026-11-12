// Package arrange renames, moves and removes notes, and lists the problems the vault checks find.
package arrange

import (
	"context"
	"errors"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/api"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/ports"
)

// UseCase arranges the notes of one vault.
type UseCase struct {
	arranger ports.NoteArranger
}

// New creates a UseCase over the arranger of one vault.
func New(arranger ports.NoteArranger) *UseCase {
	return &UseCase{arranger: arranger}
}

// Rename gives a note a new title and moves its file to the file name of the title.
func (u *UseCase) Rename(ctx context.Context, in api.NoteRenameInput) (api.NoteRenameOutput, error) {
	if strings.TrimSpace(in.Path) == "" {
		return api.NoteRenameOutput{}, errors.New("path is required")
	}
	result, err := u.arranger.RenameNote(ctx, in.Path, in.Title)
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

// Move moves notes to a folder; a note that fails does not stop the others.
func (u *UseCase) Move(ctx context.Context, in api.NoteMoveInput) (api.NoteMoveOutput, error) {
	if err := checkPaths(in.Paths); err != nil {
		return api.NoteMoveOutput{}, err
	}
	folder := strings.TrimSpace(in.Folder)
	if folder == "" {
		return api.NoteMoveOutput{}, errors.New(`folder is empty; pass "/" for the vault root`)
	}
	moves, err := u.arranger.MoveNotes(ctx, in.Paths, strings.Trim(folder, "/"))
	if err != nil {
		return api.NoteMoveOutput{}, err
	}
	out := api.NoteMoveOutput{Results: make([]api.NoteMoveItem, 0, len(moves))}
	for _, move := range moves {
		out.Results = append(out.Results, newMoveItem(move))
	}
	return out, nil
}

// Remove moves notes to the trash of the vault; a note that fails does not stop the others.
func (u *UseCase) Remove(ctx context.Context, in api.NoteRemoveInput) (api.NoteRemoveOutput, error) {
	if err := checkPaths(in.Paths); err != nil {
		return api.NoteRemoveOutput{}, err
	}
	removals, err := u.arranger.RemoveNotes(ctx, in.Paths)
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

// ListProblems runs the checks named, or the default checks, and lists what they find.
func (u *UseCase) ListProblems(ctx context.Context, in api.VaultProblemListInput) (api.VaultProblemListOutput, error) {
	problems, err := u.arranger.GetProblems(ctx, in.Checks)
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

func checkPaths(paths []string) error {
	if len(paths) == 0 {
		return errors.New("paths is required")
	}
	return nil
}

func getPathList(paths []string) []string {
	if paths == nil {
		return []string{}
	}
	return paths
}
