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

// mostFound is how many answers a client that names no number gets.
const mostFound = 20

// Titles hands the client the names in the vault that match what was typed: a
// note's own title, and the headings inside notes.
func (a *API) Titles(ctx context.Context, r *connect.Request[v1.TitlesRequest]) (*connect.Response[v1.TitlesResponse], error) {
	found, err := a.Notes.Titles(ctx, a.Vault.ID, r.Msg.GetQuery(), atMost(r.Msg.GetLimit()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	out := &v1.TitlesResponse{Found: make([]*v1.Titled, 0, len(found))}
	for _, m := range found {
		titled := &v1.Titled{
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
// Which half runs is the client's, so a client drawing what is written apart
// from what it means asks twice and each answer fills its own list.
func (a *API) Search(ctx context.Context, r *connect.Request[v1.SearchRequest]) (*connect.Response[v1.SearchResponse], error) {
	if a.Finds == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoSearching)
	}
	query := r.Msg.GetQuery()
	if strings.TrimSpace(query) == "" {
		return connect.NewResponse(&v1.SearchResponse{}), nil
	}

	found, err := a.Finds.Execute(ctx, a.Vault,
		query, search.Running(halfOf(r.Msg.GetHalf()), atMost(r.Msg.GetLimit())))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// The note each passage was read out of, asked once for the whole answer. A
	// source that is not a note is absent, and a window offering to open notes
	// offers nothing for it.
	titles, err := a.Notes.Notes(ctx, a.Vault.ID, sourcesOf(found))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	out := &v1.SearchResponse{Found: make([]*v1.Passage, 0, len(found))}
	for _, p := range found {
		// A passage is the whole window enclosing its hit, so what is drawn is the
		// words about the first run that matched.
		// A hit by meaning stands on no word, so the window opens where the chunk
		// that matched begins.
		text, at := around(p.Text, marks(p.Text, query), markdown.Counted(p.Text, p.Hit))
		passage := &v1.Passage{
			Path:     p.Source,
			Text:     text,
			At:       spansOf(at),
			Location: p.Location,
		}
		if note, held := titles[p.Source]; held {
			passage.Note = noteOf(note)
		}
		out.Found = append(out.Found, passage)
	}
	return connect.NewResponse(out), nil
}

// halfOf is the half the client named, as the use case names it.
func halfOf(half v1.Half) search.Halves {
	switch half {
	case v1.Half_HALF_WORDS:
		return search.Words
	case v1.Half_HALF_MEANING:
		return search.Meaning
	case v1.Half_HALF_UNSPECIFIED:
		return search.Both
	}
	return search.Both
}

// atMost is how many answers to give, from what the client asked for.
func atMost(limit int32) int {
	if limit <= 0 {
		return mostFound
	}
	return int(limit)
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
