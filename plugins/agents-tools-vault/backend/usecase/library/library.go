// Package library searches and reads the text of the files of a vault: notes, books, PDFs, scans and
// recordings.
package library

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/api"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/folder"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/ports"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/textplace"
)

var errNoLibrary = errors.New("the plugin was given neither text versions nor a search index")

// UseCase works the text index and the text versions of one vault. The ports of the library are nil
// when the plugin was given none, and its methods return an error.
type UseCase struct {
	files    ports.FileReader
	notes    ports.NoteLookup
	versions ports.VersionLister
	texts    ports.TextReader
	search   ports.TextSearch
}

// New creates a UseCase over the files and notes of one vault and its library.
func New(files ports.FileReader, notes ports.NoteLookup, versions ports.VersionLister, texts ports.TextReader, search ports.TextSearch) *UseCase {
	return &UseCase{files: files, notes: notes, versions: versions, texts: texts, search: search}
}

// Search returns the passages of the vault index that match the query, best first.
func (u *UseCase) Search(ctx context.Context, in api.LibrarySearchInput) (api.LibrarySearchOutput, error) {
	out := api.LibrarySearchOutput{Query: in.Query, Matches: []api.LibraryMatch{}}
	if strings.TrimSpace(in.Query) == "" {
		return out, nil
	}
	kinds := make([]domain.SourceKind, len(in.Kinds))
	for i, kind := range in.Kinds {
		kinds[i] = domain.SourceKind(kind)
	}
	if u.search == nil {
		return out, errNoLibrary
	}
	found, err := u.search.SearchText(ctx, host.SearchResultQuery{
		Query: in.Query, Limit: in.Limit, Kinds: kinds, PathPrefix: folder.GetPrefix(in.Folder),
	})
	if err != nil {
		return out, fmt.Errorf("search the library: %w", err)
	}
	titles, err := u.getTitles(ctx, found)
	if err != nil {
		return out, err
	}
	for _, result := range found {
		place := newPlace(textplace.GetSelector(result.Selector))
		if place != nil && place.CFI != "" {
			place.Chapter = result.PlaceLabel
		}
		out.Matches = append(out.Matches, api.LibraryMatch{
			Path:   result.Path,
			Title:  titles[result.Path],
			Place:  place,
			Text:   result.Text,
			Start:  result.Start,
			Length: result.Length,
			Line:   result.MatchLine + 1,
			URI:    textplace.FormatAddress(result.Path, result.Start, result.Length),
		})
	}
	return out, nil
}

func (u *UseCase) getTitles(ctx context.Context, found []host.SearchResult) (map[string]string, error) {
	titles := make(map[string]string, len(found))
	var notePaths []string
	for _, result := range found {
		base := path.Base(result.Path)
		titles[result.Path] = strings.TrimSuffix(base, path.Ext(base))
		if strings.HasSuffix(result.Path, domain.NoteExtension) {
			notePaths = append(notePaths, result.Path)
		}
	}
	refs, err := u.notes.GetNotes(ctx, notePaths)
	if err != nil {
		return nil, fmt.Errorf("read the titles of the notes: %w", err)
	}
	for notePath, ref := range refs {
		if ref.Title != "" {
			titles[notePath] = ref.Title
		}
	}
	return titles, nil
}

func newPlace(s textplace.Selector) *api.Place {
	if s.IsEmpty() {
		return nil
	}
	place := &api.Place{Page: s.Page, PageCount: 0, PositionSeconds: nil, EndSeconds: nil, CFI: s.CFI, Chapter: ""}
	if len(s.T) > 0 {
		place.PositionSeconds = &s.T[0]
	}
	if len(s.T) > 1 {
		place.EndSeconds = &s.T[1]
	}
	return place
}
