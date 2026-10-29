// Package search provides search services for the editor window.
package search

import (
	"context"
	"errors"
	"strings"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/usecase/floor"
	"github.com/jiva-studio/numen/libs/core/usecase/note"
	"github.com/jiva-studio/numen/libs/core/usecase/search"
	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/notes/markdown"
)

type VaultProvider interface {
	GetCurrentVault() domain.Vault
}

type Handler struct {
	numenv1connect.UnimplementedSearchServiceHandler

	Vaults         VaultProvider
	GetNameMatches *search.GetNameMatches
	Finds          *search.Search
	GetTitles      note.GetTitles
	GetTypes       *note.GetTypes
	// Floors is nil on a window put together without them.
	Floors *floor.Scenarios
}

func NewHandler(vaults VaultProvider) *Handler {
	return &Handler{Vaults: vaults}
}

// errNoMode is a search that named no mode to ask it in.
var errNoMode = errors.New("a search requires a mode")

// The default number of results when a client specifies none, and the maximum
// it may request. A window displays a list for reading; a larger number is a
// corpus query and is clamped to the maximum.
const (
	mostFound      = 20
	maxQueryLength = 200
)

// SearchNames hands the client the names in the vault that match what was
// typed: a note's own title, and the headings inside notes. A window standing
// on nothing holds no names.
func (h *Handler) SearchNames(
	ctx context.Context, r *connect.Request[v1.SearchNamesRequest],
) (*connect.Response[v1.SearchNamesResponse], error) {
	showing := h.Vaults.GetCurrentVault()
	if showing.ID == "" {
		return connect.NewResponse(&v1.SearchNamesResponse{}), nil
	}
	found, err := h.GetNameMatches.Execute(ctx, showing, r.Msg.GetQuery(), atMost(r.Msg.GetLimit()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	out := &v1.SearchNamesResponse{Found: make([]*v1.NameMatch, 0, len(found))}
	for _, m := range found {
		titled := &v1.NameMatch{
			Note:   &v1.Note{Path: m.Path, Title: m.Title},
			Ranges: convertUnitRangesToWire(m.Ranges),
			Type:   convertNoteTypeToWire(m.Type),
		}
		if m.Heading != "" {
			titled.Heading = &v1.Heading{Text: m.Heading, Line: int32(m.Line)}
		}
		out.Found = append(out.Found, titled)
	}
	return connect.NewResponse(out), nil
}

// SearchPassages returns the text of the vault that matches the query.
//
// The search mode is chosen by the client, so a client that displays the
// written form and the meaning separately issues two requests and fills each
// list from its own response.
//
// A window with no open vault has no text and returns no passages.
func (h *Handler) SearchPassages(
	ctx context.Context, r *connect.Request[v1.SearchPassagesRequest],
) (*connect.Response[v1.SearchPassagesResponse], error) {
	query := r.Msg.GetQuery()
	showing := h.Vaults.GetCurrentVault()
	if strings.TrimSpace(query) == "" || showing.ID == "" {
		return connect.NewResponse(&v1.SearchPassagesResponse{}), nil
	}

	mode, named := ConvertSearchModeFromWire(r.Msg.GetMode())
	if !named {
		return nil, connect.NewError(connect.CodeInvalidArgument, errNoMode)
	}

	found, err := h.Finds.Execute(ctx, showing,
		query, search.GetTypingParameters(mode, atMost(r.Msg.GetLimit())))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// The note each passage was read from, resolved once for the whole response. A
	// source that is not a note is absent, and a window offering to open notes
	// offers none for it.
	sources := getPassageSources(found)
	noted, err := h.GetTitles.Execute(ctx, showing, sources)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	titles := make(map[string]domain.NoteRef, len(noted.Notes))
	for _, ref := range noted.Notes {
		titles[ref.Path] = ref
	}
	// Which of four each of those notes is, so a passage is drawn with the mark
	// of the note it was read out of.
	types, err := h.GetTypes.Execute(ctx, showing, sources)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	out := &v1.SearchPassagesResponse{Found: make([]*v1.Passage, 0, len(found))}
	for _, p := range found {
		// A passage is the whole window enclosing its hit, so what is drawn is the
		// words about the first run that matched. A hit by meaning relies on no
		// word, and the window opens where the chunk that matched begins.
		read, matched := nearby(p.Text, p.HitAt)
		text, pos := getTextAround(read, spans(read, query), markdown.CountUTF16(read, matched))
		passage := &v1.Passage{
			Path:     p.Source,
			Text:     text,
			Ranges:   convertUnitRangesToWire(pos),
			Location: p.Location,
			Range:    &v1.Range{Start: int32(p.Start), End: int32(p.Start + p.Length)},
			Line:     int32(p.Line),
			Selector: string(p.Selector),
		}
		if noteRef, target := titles[p.Source]; target {
			passage.Note = convertNoteToWire(noteRef)
			passage.Type = convertNoteTypeToWire(types[p.Source])
		}
		out.Found = append(out.Found, passage)
	}
	return connect.NewResponse(out), nil
}

func convertNoteTypeToWire(noteType domain.NoteType) string {
	if noteType == domain.TypeNote {
		return ""
	}
	return string(noteType)
}

func convertNoteToWire(n domain.NoteRef) *v1.Note {
	return &v1.Note{Path: n.Path, Title: n.Title}
}

// ConvertSearchModeFromWire is the mode the client named, as the use case names it, and whether it
// named one at all.
func ConvertSearchModeFromWire(mode v1.SearchMode) (search.Mode, bool) {
	switch mode {
	case v1.SearchMode_SEARCH_MODE_WORDS:
		return search.Lexical, true
	case v1.SearchMode_SEARCH_MODE_MEANING:
		return search.Dense, true
	case v1.SearchMode_SEARCH_MODE_NAMES:
		return search.ByName, true
	case v1.SearchMode_SEARCH_MODE_HYBRID:
		return search.Hybrid, true
	default:
		return search.Hybrid, false
	}
}

// atMost returns the number of results to return for the limit the client requested.
func atMost(limit int32) int {
	if limit <= 0 {
		return mostFound
	}
	return min(int(limit), maxQueryLength)
}

// getPassageSources is every file the passages came out of, each named once.
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

func convertUnitRangesToWire(pos []domain.UnitRange) []*v1.Range {
	out := make([]*v1.Range, 0, len(pos))
	for _, span := range pos {
		out = append(out, &v1.Range{Start: int32(span.Start), End: int32(span.End)})
	}
	return out
}
