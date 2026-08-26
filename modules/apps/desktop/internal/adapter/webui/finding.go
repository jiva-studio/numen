package webui

import (
	"context"
	"errors"
	"strings"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/markdown"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/search"
)

// errNoSearching is what a build with nothing to search the text with answers.
var errNoSearching = errors.New("this build cannot search the text of a vault")

// How many answers a client gets when it names no number, and the most it may
// ask for. A window draws a list a person reads; a number past that is a
// question about the corpus and is answered as the ceiling.
const (
	mostFound = 20
	mostAsked = 200
)

// Names hands the client the names in the vault that match what was typed: a
// note's own title, and the headings inside notes. A window standing on nothing
// holds no names.
func (a *API) Names(ctx context.Context, r *connect.Request[v1.NamesRequest]) (*connect.Response[v1.NamesResponse], error) {
	showing := a.Showing()
	if showing.ID == "" {
		return connect.NewResponse(&v1.NamesResponse{}), nil
	}
	found, err := a.Notes.Names(ctx, showing.ID, r.Msg.GetQuery(), atMost(r.Msg.GetLimit()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	out := &v1.NamesResponse{Found: make([]*v1.Named, 0, len(found))}
	for _, m := range found {
		titled := &v1.Named{
			Note: &v1.Note{Path: m.Path, Title: m.Title},
			At:   spansOf(m.At),
		}
		if m.Heading != "" {
			titled.Heading = &v1.Heading{Text: m.Heading, Line: int32(m.Line)}
		}
		out.Found = append(out.Found, titled)
	}
	return connect.NewResponse(out), nil
}

// Search hands the client the text the vault holds that answers what was typed.
//
// Which way it is asked is the client's, so a client drawing what is written apart
// from what it means asks twice and each answer fills its own list.
//
// A window standing on nothing holds no text.
func (a *API) Search(ctx context.Context, r *connect.Request[v1.SearchRequest]) (*connect.Response[v1.SearchResponse], error) {
	if a.Finds == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoSearching)
	}
	query := r.Msg.GetQuery()
	showing := a.Showing()
	if strings.TrimSpace(query) == "" || showing.ID == "" {
		return connect.NewResponse(&v1.SearchResponse{}), nil
	}

	found, err := a.Finds.Execute(ctx, showing,
		query, search.Typing(wayOf(r.Msg.GetWay()), atMost(r.Msg.GetLimit())))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// The note each passage was read out of, asked once for the whole answer. A
	// source that is not a note is absent, and a window offering to open notes
	// offers nothing for it.
	titles, err := a.Notes.Notes(ctx, showing.ID, sourcesOf(found))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	out := &v1.SearchResponse{Found: make([]*v1.Passage, 0, len(found))}
	for _, p := range found {
		// A passage is the whole window enclosing its hit, so what is drawn is the
		// words about the first run that matched. A hit by meaning stands on no
		// word, and the window opens where the chunk that matched begins.
		read, hit := nearby(p.Text, p.HitAt)
		text, at := around(read, marks(read, query), markdown.Counted(read, hit))
		passage := &v1.Passage{
			Path:     p.Source,
			Text:     text,
			At:       spansOf(at),
			Location: p.Location,
			Start:    int32(p.Start),
			Length:   int32(p.Length),
			Line:     int32(p.Line),
		}
		if note, held := titles[p.Source]; held {
			passage.Note = noteOf(note)
		}
		out.Found = append(out.Found, passage)
	}
	return connect.NewResponse(out), nil
}

// wayOf is the way the client named, as the use case names it.
func wayOf(way v1.Way) search.Way {
	switch way {
	case v1.Way_WAY_WORDS:
		return search.ByWords
	case v1.Way_WAY_MEANING:
		return search.ByMeaning
	case v1.Way_WAY_NAMES:
		return search.ByName
	case v1.Way_WAY_UNSPECIFIED:
		return search.EveryWay
	}
	return search.EveryWay
}

// atMost is how many answers to give, from what the client asked for.
func atMost(limit int32) int {
	if limit <= 0 {
		return mostFound
	}
	return min(int(limit), mostAsked)
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

func spansOf(at []domain.Span) []*v1.Span {
	out := make([]*v1.Span, 0, len(at))
	for _, span := range at {
		out = append(out, &v1.Span{From: int32(span.From), To: int32(span.To)})
	}
	return out
}
