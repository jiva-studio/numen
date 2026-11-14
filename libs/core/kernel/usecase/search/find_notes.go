package search

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// TabKinds finds the kind of tab each of the notes at some paths opens in.
type TabKinds interface {
	Execute(ctx context.Context, v domain.Vault, paths []string) (map[string]string, error)
}

// NameResults are the names that match what was typed and the kind of tab each note opens in, by path.
type NameResults struct {
	Matches  []domain.NameMatch
	TabKinds map[string]string
}

// FindNames finds the names in a vault that match the words typed, with the kind of tab each note opens in.
type FindNames struct {
	matches  *GetNameMatches
	tabKinds TabKinds
}

// NewFindNames creates the search over the names of the index.
func NewFindNames(matches noteport.NameMatches, tabKinds TabKinds) *FindNames {
	return &FindNames{matches: NewGetNameMatches(matches), tabKinds: tabKinds}
}

// The default number of results when a client specifies none, and the maximum
// it may request. A window displays a list for reading; a larger number is a
// corpus query and is clamped to the maximum.
const (
	MostFound      = 20
	maxWindowLimit = 200
)

func clampWindowLimit(limit int) int {
	if limit <= 0 {
		return MostFound
	}
	return min(limit, maxWindowLimit)
}

func (u *FindNames) Execute(ctx context.Context, v domain.Vault, query string, limit int) (NameResults, error) {
	found, err := u.matches.Execute(ctx, v, query, clampWindowLimit(limit))
	if err != nil {
		return NameResults{}, err
	}
	paths := make([]string, 0, len(found))
	for _, m := range found {
		paths = append(paths, m.Path)
	}
	kinds, err := u.tabKinds.Execute(ctx, v, paths)
	if err != nil {
		return NameResults{}, err
	}
	return NameResults{Matches: found, TabKinds: kinds}, nil
}

// FindTextOutput are the search results found and, for the notes they were read from, what each note is
// and the kind of tab it opens in, by path. A source that is not a note is absent from the maps.
type FindTextOutput struct {
	Results []domain.SearchResult
	Notes    map[string]domain.NoteRef
	Types    map[string]domain.NoteType
	TabKinds map[string]string
}

// FindText searches the text of a vault in one mode and resolves the notes the search results were read from.
type FindText struct {
	finds    resultFinding
	notes    noteport.NoteRefs
	types    noteport.NoteTypes
	tabKinds TabKinds
}

// NewFindText creates the search over the text of the vault and the notes of the index.
func NewFindText(finds Search, notes noteport.NoteRefs, types noteport.NoteTypes, tabKinds TabKinds) *FindText {
	return &FindText{finds: finds, notes: notes, types: types, tabKinds: tabKinds}
}

// Execute searches the way `mode` names while the query is being typed, and returns at most `limit` search results.
func (u *FindText) Execute(ctx context.Context, v domain.Vault, query string, mode Mode, limit int) (FindTextOutput, error) {
	found, err := u.finds.Execute(ctx, v, query, GetTypingParameters(mode, clampWindowLimit(limit)))
	if err != nil {
		return FindTextOutput{}, err
	}
	sources := getResultPaths(found)
	notes, err := u.notes.GetNotes(ctx, v.ID, sources)
	if err != nil {
		return FindTextOutput{}, err
	}
	types, err := u.types.GetTypes(ctx, v.ID, sources)
	if err != nil {
		return FindTextOutput{}, err
	}
	kinds, err := u.tabKinds.Execute(ctx, v, sources)
	if err != nil {
		return FindTextOutput{}, err
	}
	return FindTextOutput{Results: found, Notes: notes, Types: types, TabKinds: kinds}, nil
}

func getResultPaths(found []domain.SearchResult) []string {
	observed := make(map[string]bool, len(found))
	out := make([]string, 0, len(found))
	for _, p := range found {
		if observed[p.Path] {
			continue
		}
		observed[p.Path] = true
		out = append(out, p.Path)
	}
	return out
}

// resultFinding searches the text of a vault.
type resultFinding interface {
	Execute(ctx context.Context, v domain.Vault, query string, p Parameters) ([]domain.SearchResult, error)
}
