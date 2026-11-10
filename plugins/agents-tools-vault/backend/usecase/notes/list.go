package notes

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/api"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/folder"
	"golang.org/x/text/unicode/norm"
)

const (
	defaultListLimit = 100
	maxListLimit     = 500
	maxTitleBatch    = 50
)

// List returns one page of the notes of the vault or of a folder, ordered by path.
func (u *UseCase) List(ctx context.Context, in api.NoteListInput) (api.NoteListOutput, error) {
	limit, err := getListLimit(in)
	if err != nil {
		return api.NoteListOutput{}, err
	}
	paths, err := u.lookup.GetPathsOfType(ctx, domain.TypeNote)
	if err != nil {
		return api.NoteListOutput{}, err
	}
	if prefix := folder.GetPrefix(in.Folder); prefix != "" {
		composed := norm.NFC.String(prefix)
		paths = slices.DeleteFunc(paths, func(p string) bool { return !strings.HasPrefix(norm.NFC.String(p), composed) })
	}
	slices.Sort(paths)

	total := len(paths)
	start := min(in.Offset, total)
	end := min(start+limit, total)
	page := paths[start:end]
	refs, err := u.lookup.GetNotes(ctx, page)
	if err != nil {
		return api.NoteListOutput{}, err
	}
	notes := make([]api.NoteEntry, 0, len(page))
	for _, p := range page {
		notes = append(notes, api.NoteEntry{Path: p, Title: refs[p].Title})
	}
	next := 0
	if end < total {
		next = end
	}
	return api.NoteListOutput{Notes: notes, Total: total, NextOffset: next}, nil
}

func getListLimit(in api.NoteListInput) (int, error) {
	limit := in.Limit
	switch {
	case limit < 0:
		return 0, errors.New("limit must not be negative")
	case limit == 0:
		limit = defaultListLimit
	case limit > maxListLimit:
		return 0, fmt.Errorf("limit %d is above the maximum of %d", limit, maxListLimit)
	}
	if in.Offset < 0 {
		return 0, errors.New("offset must not be negative")
	}
	return limit, nil
}

// ListTitles returns the title and identifier of each note named by path, and the paths that name no note.
func (u *UseCase) ListTitles(ctx context.Context, in api.NoteTitleListInput) (api.NoteTitleListOutput, error) {
	if len(in.Paths) > maxTitleBatch {
		return api.NoteTitleListOutput{}, fmt.Errorf("%d paths given, at most %d are allowed", len(in.Paths), maxTitleBatch)
	}
	refs, err := u.lookup.GetNotes(ctx, in.Paths)
	if err != nil {
		return api.NoteTitleListOutput{}, err
	}
	out := api.NoteTitleListOutput{Notes: make([]api.NoteTitle, 0, len(refs)), MissingPaths: []string{}}
	for _, p := range in.Paths {
		ref, ok := refs[p]
		if !ok {
			out.MissingPaths = append(out.MissingPaths, p)
			continue
		}
		out.Notes = append(out.Notes, api.NoteTitle{Path: p, Title: ref.Title, ID: ref.ID})
	}
	return out, nil
}

// ResolveName returns the paths of the notes with a file name, which is what a wikilink names.
func (u *UseCase) ResolveName(ctx context.Context, in api.NoteResolveInput) (api.NoteResolveOutput, error) {
	if strings.TrimSpace(in.Name) == "" {
		return api.NoteResolveOutput{}, errors.New("name is required")
	}
	paths, err := u.lookup.GetNamedPaths(ctx, in.Name)
	if err != nil {
		return api.NoteResolveOutput{}, fmt.Errorf("%s cannot be resolved: %w", in.Name, err)
	}
	if paths == nil {
		paths = []string{}
	}
	return api.NoteResolveOutput{Paths: paths}, nil
}
