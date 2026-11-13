package query

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// TabKinds finds the kind of tab each of the notes at some paths opens in.
type TabKinds interface {
	Execute(ctx context.Context, v domain.Vault, paths []string) (map[string]string, error)
}

// RelatedNotesView is a note with the notes joined to it, what each of them is and the kind of tab each
// opens in, by path.
type RelatedNotesView struct {
	domain.NoteGraph
	Types    map[string]domain.NoteType
	TabKinds map[string]string
}

// GetRelatedNotesView gathers the related notes of a note with the type and the kind of tab of each note in it.
type GetRelatedNotesView struct {
	relatedNotes ListRelatedNotes
	types        noteport.NoteTypes
	tabKinds     TabKinds
}

// NewGetRelatedNotesView creates the gathering over the links and the notes of the index.
func NewGetRelatedNotesView(links noteport.LinkQueries, notes RefQueries, types noteport.NoteTypes, tabKinds TabKinds) *GetRelatedNotesView {
	return &GetRelatedNotesView{relatedNotes: NewListRelatedNotes(links, notes), types: types, tabKinds: tabKinds}
}

// Execute gathers the related notes of the note at the path. Depth 2 adds the notes related to each of them. The types are read once for the whole graph.
func (u *GetRelatedNotesView) Execute(ctx context.Context, v domain.Vault, path string, depth uint32) (RelatedNotesView, error) {
	found, err := u.relatedNotes.Execute(ctx, v, path, depth)
	if err != nil {
		return RelatedNotesView{}, err
	}
	paths := []string{found.Focus.Path}
	for _, related := range found.Related {
		paths = append(paths, related.Path)
	}
	types, err := u.types.GetTypes(ctx, v.ID, paths)
	if err != nil {
		return RelatedNotesView{}, err
	}
	tabKinds, err := u.tabKinds.Execute(ctx, v, paths)
	if err != nil {
		return RelatedNotesView{}, err
	}
	return RelatedNotesView{NoteGraph: found, Types: types, TabKinds: tabKinds}, nil
}

// NoteHeadings is the headings of the note at a path, in the order they stand in the note.
type NoteHeadings struct {
	Path     string
	Headings []domain.Heading
}

// ListHeadings lists what each of the notes at some paths is divided into.
type ListHeadings struct {
	queries noteport.Headings
}

// NewListHeadings creates the listing over the headings of the index.
func NewListHeadings(queries noteport.Headings) *ListHeadings {
	return &ListHeadings{queries: queries}
}

// Execute lists the notes in the order the paths name them, and leaves out a path that names no note.
func (u *ListHeadings) Execute(ctx context.Context, v domain.Vault, paths []string) ([]NoteHeadings, error) {
	found, err := u.queries.GetHeadings(ctx, v.ID, paths)
	if err != nil {
		return nil, err
	}
	out := make([]NoteHeadings, 0, len(found))
	for _, path := range paths {
		if headings, ok := found[path]; ok {
			out = append(out, NoteHeadings{Path: path, Headings: headings})
		}
	}
	return out, nil
}
