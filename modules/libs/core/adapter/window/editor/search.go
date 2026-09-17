package editor

import (
	"context"
	"errors"
	"strings"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/markdown"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/search"
)

// errNoMode is a search that named no mode to ask it in.
var errNoMode = errors.New("a search says how it is asked")

// errTooManyPaths is a filter naming more paths than are answered at once. The
// whole filter is refused, and no part of it is answered.
var errTooManyPaths = errors.New("that is more paths than are answered at once")

// How many answers a client gets when it names no number, and the most it may
// ask for. A window draws a list a person reads; a number past that is a
// question about the corpus and is answered as the ceiling.
const (
	mostFound = 20
	mostAsked = 200
)

// SearchNames hands the client the names in the vault that match what was
// typed: a note's own title, and the headings inside notes. A window standing
// on nothing holds no names.
func (a *API) SearchNames(
	ctx context.Context, r *connect.Request[v1.SearchNamesRequest],
) (*connect.Response[v1.SearchNamesResponse], error) {
	showing := a.GetShownVault()
	if showing.ID == "" {
		return connect.NewResponse(&v1.SearchNamesResponse{}), nil
	}
	found, err := a.Notes.Queries.Names(ctx, showing.ID, r.Msg.GetQuery(), atMost(r.Msg.GetLimit()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	out := &v1.SearchNamesResponse{Found: make([]*v1.NameMatch, 0, len(found))}
	for _, m := range found {
		titled := &v1.NameMatch{
			Note:  &v1.Note{Path: m.Path, Title: m.Title},
			Spans: unitSpansOf(m.Spans),
			Type:  typeOf(m.Type),
		}
		if m.Heading != "" {
			titled.Heading = &v1.Heading{Text: m.Heading, Line: int32(m.Line)}
		}
		out.Found = append(out.Found, titled)
	}
	return connect.NewResponse(out), nil
}

// ListHeadings hands the client the headings of the notes the filter names. A
// window standing on nothing holds no note to divide.
func (a *API) ListHeadings(
	ctx context.Context,
	r *connect.Request[v1.ListHeadingsRequest],
) (*connect.Response[v1.ListHeadingsResponse], error) {
	showing := a.GetShownVault()
	if showing.ID == "" {
		return connect.NewResponse(&v1.ListHeadingsResponse{}), nil
	}
	paths, err := eachOnce(r.Msg.GetPaths())
	if err != nil {
		return nil, err
	}
	found, err := a.Notes.Queries.Headings(ctx, showing.ID, paths)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// In the order the filter named them.
	out := &v1.ListHeadingsResponse{Headings: make([]*v1.NoteHeadings, 0, len(found))}
	for _, path := range paths {
		headings, held := found[path]
		if !held {
			continue
		}
		one := &v1.NoteHeadings{Path: path, Headings: make([]*v1.Heading, 0, len(headings))}
		for _, h := range headings {
			one.Headings = append(one.Headings, &v1.Heading{
				Text:  h.Text,
				Line:  int32(h.Line),
				Level: int32(h.Level),
			})
		}
		out.Headings = append(out.Headings, one)
	}
	return connect.NewResponse(out), nil
}

// SearchPassages hands the client the text the vault holds that answers what
// was typed.
//
// Which way it is asked is the client's, so a client drawing what is written apart
// from what it means asks twice and each answer fills its own list.
//
// A window standing on nothing holds no text.
func (a *API) SearchPassages(
	ctx context.Context, r *connect.Request[v1.SearchPassagesRequest],
) (*connect.Response[v1.SearchPassagesResponse], error) {
	query := r.Msg.GetQuery()
	showing := a.GetShownVault()
	if strings.TrimSpace(query) == "" || showing.ID == "" {
		return connect.NewResponse(&v1.SearchPassagesResponse{}), nil
	}

	mode, named := modeOf(r.Msg.GetMode())
	if !named {
		return nil, connect.NewError(connect.CodeInvalidArgument, errNoMode)
	}

	found, err := a.Finds.Execute(ctx, showing,
		query, search.GetTypingParameters(mode, atMost(r.Msg.GetLimit())))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// The note each passage was read out of, asked once for the whole answer. A
	// source that is not a note is absent, and a window offering to open notes
	// offers nothing for it.
	sources := sourcesOf(found)
	titles, err := a.Notes.Queries.Notes(ctx, showing.ID, sources)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	// Which of four each of those notes is, so a passage is drawn with the mark
	// of the note it was read out of.
	types, err := a.Notes.Queries.Types(ctx, showing.ID, sources)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	out := &v1.SearchPassagesResponse{Found: make([]*v1.Passage, 0, len(found))}
	for _, p := range found {
		// A passage is the whole window enclosing its hit, so what is drawn is the
		// words about the first run that matched. A hit by meaning stands on no
		// word, and the window opens where the chunk that matched begins.
		read, hit := nearby(p.Text, p.HitAt)
		text, at := getTextAround(read, spans(read, query), markdown.CountUTF16(read, hit))
		passage := &v1.Passage{
			Path:     p.Source,
			Text:     text,
			Spans:    unitSpansOf(at),
			Location: p.Location,
			Span:     &v1.Span{From: int32(p.Start), To: int32(p.Start + p.Length)},
			Line:     int32(p.Line),
			Kind:     kindOf(p.Kind),
		}
		if note, held := titles[p.Source]; held {
			passage.Note = noteOf(note)
			passage.Type = typeOf(types[p.Source])
		}
		out.Found = append(out.Found, passage)
	}
	return connect.NewResponse(out), nil
}

// modeOf is the mode the client named, as the use case names it, and whether it
// named one at all.
func modeOf(mode v1.SearchMode) (search.Mode, bool) {
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

// atMost is how many answers to give, from what the client asked for.
func atMost(limit int32) int {
	if limit <= 0 {
		return mostFound
	}
	return min(int(limit), mostAsked)
}

// eachOnce is the paths a client asked about, each named once. A filter past
// the ceiling is refused: each path costs a round trip of its own, and an
// answer cut to fit says nothing about which paths were left out.
func eachOnce(paths []string) ([]string, error) {
	seen := make(map[string]bool, len(paths))
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		if seen[path] {
			continue
		}
		seen[path] = true
		out = append(out, path)
	}
	if len(out) > mostAsked {
		return nil, connect.NewError(connect.CodeInvalidArgument, errTooManyPaths)
	}
	return out, nil
}

// sourcesOf is every file the passages came out of, each named once.
func sourcesOf(found []domain.Passage) []string {
	seen := make(map[string]bool, len(found))
	out := make([]string, 0, len(found))
	for _, p := range found {
		if seen[p.Source] {
			continue
		}
		seen[p.Source] = true
		out = append(out, p.Source)
	}
	return out
}

// byteSpansOf and unitSpansOf both write runs onto the wire. They are two
// functions because the runs are counted in two different things, and the field
// the answer carries them in says which.
func byteSpansOf(at []domain.ByteSpan) []*v1.Span {
	out := make([]*v1.Span, 0, len(at))
	for _, span := range at {
		out = append(out, &v1.Span{From: int32(span.From), To: int32(span.To)})
	}
	return out
}

func unitSpansOf(at []domain.UnitSpan) []*v1.Span {
	out := make([]*v1.Span, 0, len(at))
	for _, span := range at {
		out = append(out, &v1.Span{From: int32(span.From), To: int32(span.To)})
	}
	return out
}
