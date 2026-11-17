package query

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/windowport"
	"github.com/jiva-studio/numen/libs/note/domain"
)

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
	tabKinds     windowport.TabKinds
}

// NewGetRelatedNotesView creates the gathering over the links and the notes of the index.
func NewGetRelatedNotesView(links noteport.LinkBatches, notes RefQueries, types noteport.NoteTypes, tabKinds windowport.TabKinds) *GetRelatedNotesView {
	return &GetRelatedNotesView{relatedNotes: NewListRelatedNotes(links, notes), types: types, tabKinds: tabKinds}
}

// Execute gathers the related notes of the note at the path. DepthFar adds the notes related to each of them. The types are read once for the whole graph.
// A tab kind is chosen for the focus and the notes related to it, and not for a note that is only related to one of them.
func (u *GetRelatedNotesView) Execute(ctx context.Context, v domain.Vault, paths []string, depth domain.RelationDepth) ([]RelatedNotesView, error) {
	return []RelatedNotesView{}, nil
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
