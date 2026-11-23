package query

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/windowport"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
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

// Execute gathers the related notes of the notes at the paths, one view for each path in the order of the paths. DepthFar adds the notes related to each of them. The types are read once for all the graphs, and so are the tab kinds.
// A tab kind is chosen for a focus and the notes related to it, and not for a note that is only related to one of them.
func (u *GetRelatedNotesView) Execute(ctx context.Context, v domain.Vault, paths []string, depth domain.RelationDepth) ([]RelatedNotesView, error) {
	if len(paths) > domain.MaxRelatedNotePaths {
		return nil, fmt.Errorf("%w: at most %d, not %d", domain.ErrTooManyRelatedPaths, domain.MaxRelatedNotePaths, len(paths))
	}
	graphs, err := u.relatedNotes.Execute(ctx, v, paths, depth)
	if err != nil {
		return nil, err
	}
	if len(graphs) == 0 {
		return []RelatedNotesView{}, nil
	}
	var all, near []string
	for _, found := range graphs {
		if found.Focus.Path == "" {
			continue
		}
		all = append(all, found.Focus.Path)
		near = append(near, found.Focus.Path)
		for _, related := range found.Related {
			all = append(all, related.Path)
			if related.RelatedTo == "" {
				near = append(near, related.Path)
			}
		}
	}
	types, err := u.types.GetTypes(ctx, v.ID, uniquePaths(all))
	if err != nil {
		return nil, err
	}
	tabKinds, err := u.tabKinds.ListTabKinds(ctx, v, uniquePaths(near))
	if err != nil {
		return nil, err
	}
	out := make([]RelatedNotesView, len(graphs))
	for i, found := range graphs {
		out[i] = RelatedNotesView{NoteGraph: found, Types: types, TabKinds: tabKinds}
	}
	return out, nil
}

// NoteHeadings is the headings of the note at a path, in the order they stand in the note.
type NoteHeadings struct {
	Path     string
	Headings []domain.Heading
}

// ListHeadings lists what each of the notes at some paths is divided into.
type ListHeadings struct {
	queries Headings
}

// NewListHeadings creates the listing over the headings of the index.
func NewListHeadings(queries Headings) *ListHeadings {
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
