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

// PassageResults are the passages found and, for the notes they were read from, what each note is
// and the kind of tab it opens in, by path. A source that is not a note is absent from the maps.
type PassageResults struct {
	Passages []domain.Passage
	Notes    map[string]domain.NoteRef
	Types    map[string]domain.NoteType
	TabKinds map[string]string
}

// FindPassages searches the text of a vault in one mode and resolves the notes the passages were read from.
type FindPassages struct {
	finds    passageFinding
	notes    noteport.NoteRefs
	types    noteport.NoteTypes
	tabKinds TabKinds
}

// NewFindPassages creates the search over the text of the vault and the notes of the index.
func NewFindPassages(finds Search, notes noteport.NoteRefs, types noteport.NoteTypes, tabKinds TabKinds) *FindPassages {
	return &FindPassages{finds: finds, notes: notes, types: types, tabKinds: tabKinds}
}

// Execute searches the way `mode` names while the query is being typed, and returns at most `limit` passages.
func (u *FindPassages) Execute(ctx context.Context, v domain.Vault, query string, mode Mode, limit int) (PassageResults, error) {
	found, err := u.finds.Execute(ctx, v, query, GetTypingParameters(mode, clampWindowLimit(limit)))
	if err != nil {
		return PassageResults{}, err
	}
	sources := getPassageSources(found)
	notes, err := u.notes.GetNotes(ctx, v.ID, sources)
	if err != nil {
		return PassageResults{}, err
	}
	types, err := u.types.GetTypes(ctx, v.ID, sources)
	if err != nil {
		return PassageResults{}, err
	}
	kinds, err := u.tabKinds.Execute(ctx, v, sources)
	if err != nil {
		return PassageResults{}, err
	}
	return PassageResults{Passages: found, Notes: notes, Types: types, TabKinds: kinds}, nil
}

func getPassageSources(found []domain.Passage) []string {
	observed := make(map[string]bool, len(found))
	out := make([]string, 0, len(found))
	for _, p := range found {
		if observed[p.Source] {
			continue
		}
		observed[p.Source] = true
		out = append(out, p.Source)
	}
	return out
}

// passageFinding searches the text of a vault.
type passageFinding interface {
	Execute(ctx context.Context, v domain.Vault, query string, p Parameters) ([]domain.Passage, error)
}
