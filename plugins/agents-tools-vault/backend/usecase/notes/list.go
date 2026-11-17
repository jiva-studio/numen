package notes

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/folder"
	"golang.org/x/text/unicode/norm"
)

const (
	defaultListLimit = 100
	maxListLimit     = 500
	maxTitleBatch    = 50
)

// NoteTitle is the path, title and identifier of a note; ID is empty for a note without one.
type NoteTitle struct {
	Path  string
	Title string
	ID    string
}

// NotePage is one page of the notes of the vault: the notes, the number of notes that match, and the
// offset of the next page, 0 on the last page. The notes have no ID.
type NotePage struct {
	Notes      []NoteTitle
	Total      int
	NextOffset int
}

// List returns one page of the notes of the vault or of a folder, ordered by path.
func (u *UseCase) List(ctx context.Context, folderName string, limitGiven, offset int) (NotePage, error) {
	limit, err := getListLimit(limitGiven, offset)
	if err != nil {
		return NotePage{}, err
	}
	paths, err := u.lookup.GetPathsOfType(ctx, domain.TypeNote)
	if err != nil {
		return NotePage{}, err
	}
	if prefix := folder.GetPrefix(folderName); prefix != "" {
		composed := norm.NFC.String(prefix)
		paths = slices.DeleteFunc(paths, func(p string) bool { return !strings.HasPrefix(norm.NFC.String(p), composed) })
	}
	slices.Sort(paths)

	total := len(paths)
	start := min(offset, total)
	end := min(start+limit, total)
	page := paths[start:end]
	refs, err := u.lookup.GetNotes(ctx, page)
	if err != nil {
		return NotePage{}, err
	}
	notes := make([]NoteTitle, 0, len(page))
	for _, p := range page {
		notes = append(notes, NoteTitle{Path: p, Title: refs[p].Title, ID: ""})
	}
	next := 0
	if end < total {
		next = end
	}
	return NotePage{Notes: notes, Total: total, NextOffset: next}, nil
}

func getListLimit(limit, offset int) (int, error) {
	switch {
	case limit < 0:
		return 0, errors.New("limit must not be negative")
	case limit == 0:
		limit = defaultListLimit
	case limit > maxListLimit:
		return 0, fmt.Errorf("limit %d is above the maximum of %d", limit, maxListLimit)
	}
	if offset < 0 {
		return 0, errors.New("offset must not be negative")
	}
	return limit, nil
}

// ListTitles returns the title and identifier of each note named by path, and the paths that name no note.
func (u *UseCase) ListTitles(ctx context.Context, paths []string) (found []NoteTitle, missing []string, err error) {
	if len(paths) > maxTitleBatch {
		return nil, nil, fmt.Errorf("%d paths given, at most %d are allowed", len(paths), maxTitleBatch)
	}
	refs, err := u.lookup.GetNotes(ctx, paths)
	if err != nil {
		return nil, nil, err
	}
	found, missing = make([]NoteTitle, 0, len(refs)), []string{}
	for _, p := range paths {
		ref, ok := refs[p]
		if !ok {
			missing = append(missing, p)
			continue
		}
		found = append(found, NoteTitle{Path: p, Title: ref.Title, ID: ref.ID})
	}
	return found, missing, nil
}

// ResolveName returns the paths of the notes with a file name, which is what a wikilink names.
func (u *UseCase) ResolveName(ctx context.Context, name string) ([]string, error) {
	if strings.TrimSpace(name) == "" {
		return nil, errors.New("name is required")
	}
	paths, err := u.lookup.GetNamedPaths(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("%s cannot be resolved: %w", name, err)
	}
	if paths == nil {
		paths = []string{}
	}
	return paths, nil
}
