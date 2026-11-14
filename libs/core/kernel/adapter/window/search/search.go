// Package search provides search services for the editor window.
package search

import (
	"context"
	"errors"
	"strings"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/searchcancel"
	"github.com/jiva-studio/numen/libs/note/domain"
)

type VaultProvider interface {
	GetCurrentVault() domain.Vault
}

// Handler answers SearchService over the shown vault.
type Handler struct {
	numenv1connect.UnimplementedSearchServiceHandler

	vaults    VaultProvider
	findNames *search.FindNames
	findText  *search.FindText
	searches  *searchcancel.Searches
}

// NewHandler is the handler of SearchService over the names and the text of the shown vault.
func NewHandler(
	vaults VaultProvider, names *search.FindNames, texts *search.FindText, searches *searchcancel.Searches,
) *Handler {
	return &Handler{vaults: vaults, findNames: names, findText: texts, searches: searches}
}

// errNoMode is a search that named no mode to ask it in.
var errNoMode = errors.New("a search requires a mode")

// SearchNames hands the client the names in the vault that match what was
// typed: a note's own title, and the headings inside notes. A window standing
// on nothing holds no names.
func (h *Handler) SearchNames(
	ctx context.Context, r *connect.Request[v1.SearchNamesRequest],
) (*connect.Response[v1.SearchNamesResponse], error) {
	showing := h.vaults.GetCurrentVault()
	if showing.ID == "" {
		return connect.NewResponse(&v1.SearchNamesResponse{}), nil
	}
	ctx, done := h.searches.Start(ctx, r.Msg.GetSearchId())
	defer done()
	if err := ctx.Err(); err != nil {
		return nil, connect.NewError(connect.CodeCanceled, err)
	}
	found, err := h.findNames.Execute(ctx, showing, r.Msg.GetQuery(), int(r.Msg.GetLimit()))
	if errors.Is(err, context.Canceled) {
		return nil, connect.NewError(connect.CodeCanceled, err)
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	tabKinds := found.TabKinds
	out := &v1.SearchNamesResponse{Found: make([]*v1.NameMatch, 0, len(found.Matches))}
	for _, m := range found.Matches {
		titled := &v1.NameMatch{
			Note:    &v1.Note{Path: m.Path, Title: m.Title},
			Ranges:  convertUnitRangesToWire(m.Ranges),
			Type:    convertNoteTypeToWire(m.Type),
			TabKind: tabKinds[m.Path],
		}
		if m.Heading != "" {
			titled.Heading = &v1.Heading{Text: m.Heading, Line: int32(m.Line)}
		}
		out.Found = append(out.Found, titled)
	}
	return connect.NewResponse(out), nil
}

// SearchText returns the text of the vault that matches the query.
// The client chooses the search mode, so a client that shows the written form
// and the meaning separately issues two requests, one per list.
// A window with no open vault has no text and returns no search results.
func (h *Handler) SearchText(
	ctx context.Context, r *connect.Request[v1.SearchTextRequest],
) (*connect.Response[v1.SearchTextResponse], error) {
	query := r.Msg.GetQuery()
	showing := h.vaults.GetCurrentVault()
	if strings.TrimSpace(query) == "" || showing.ID == "" {
		return connect.NewResponse(&v1.SearchTextResponse{}), nil
	}

	mode, named := ConvertSearchModeFromWire(r.Msg.GetMode())
	if !named {
		return nil, connect.NewError(connect.CodeInvalidArgument, errNoMode)
	}

	ctx, done := h.searches.Start(ctx, r.Msg.GetSearchId())
	defer done()
	if err := ctx.Err(); err != nil {
		return nil, connect.NewError(connect.CodeCanceled, err)
	}
	found, err := h.findText.Execute(ctx, showing, query, mode, int(r.Msg.GetLimit()))
	if errors.Is(err, context.Canceled) {
		return nil, connect.NewError(connect.CodeCanceled, err)
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	out := &v1.SearchTextResponse{Found: make([]*v1.SearchResult, 0, len(found.Results))}
	for _, p := range found.Results {
		text, pos := search.CutSnippet(p.Text, p.MatchOffset, query)
		result := &v1.SearchResult{
			Path:        p.Path,
			Text:        text,
			MatchRanges: convertUnitRangesToWire(pos),
			PlaceLabel:  p.PlaceLabel,
			ByteRange:   &v1.Range{Start: int32(p.Start), End: int32(p.Start + p.Length)},
			MatchLine:   int32(p.MatchLine),
			Selector:    string(p.Selector),
		}
		// A source that is not a note is absent, and a window offering to open notes offers none for it.
		if noteRef, target := found.Notes[p.Path]; target {
			result.Note = convertNoteToWire(noteRef)
			result.Type = convertNoteTypeToWire(found.Types[p.Path])
			result.TabKind = found.TabKinds[p.Path]
		}
		out.Found = append(out.Found, result)
	}
	return connect.NewResponse(out), nil
}

// CancelSearch cancels the context of the SearchNames or SearchText call that carries the search id.
// The window sends it when it aborts the call, because an aborted request does not end the context of
// the call.
func (h *Handler) CancelSearch(
	_ context.Context, r *connect.Request[v1.CancelSearchRequest],
) (*connect.Response[v1.CancelSearchResponse], error) {
	h.searches.Cancel(r.Msg.GetSearchId())
	return connect.NewResponse(&v1.CancelSearchResponse{}), nil
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
		return search.Sources, true
	case v1.SearchMode_SEARCH_MODE_NAMES:
		return search.ByName, true
	case v1.SearchMode_SEARCH_MODE_HYBRID:
		return search.Hybrid, true
	default:
		return search.Hybrid, false
	}
}

func convertUnitRangesToWire(pos []domain.UnitRange) []*v1.Range {
	out := make([]*v1.Range, 0, len(pos))
	for _, span := range pos {
		out = append(out, &v1.Range{Start: int32(span.Start), End: int32(span.End)})
	}
	return out
}
