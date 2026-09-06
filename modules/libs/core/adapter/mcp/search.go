package mcp

import (
	"context"
	"fmt"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/search"
)

// maxMatches is how many passages one call may ask for. A call asking for more
// is refused and told so.
const maxMatches = 100

// passagesEach is how many places in one file a search answers with. A book
// speaks about a thing in several places, and a reader who cannot turn the page
// is told about all of them.
const passagesEach = 3

// Passage is what a search returns: the text around a hit, and where it came
// from. A hit inside a book names the book, and a hit inside a note names the
// note.
type Passage struct {
	Source   string `json:"source" jsonschema:"the file the text is read from, relative to the vault folder"`
	Location string `json:"location,omitempty" jsonschema:"where this sits in the source's own numbering — a chapter, a printed page — absent when the format offered none"`
	Text     string `json:"text" jsonschema:"the passage itself"`
	Start    int    `json:"start" jsonschema:"where the passage begins in the source's text, in bytes; hand it to source_focus to put this place in front of the person"`
	Length   int    `json:"length" jsonschema:"how long the passage is, in bytes"`
}

// addNoteSearch is what the vault holds that answers what somebody typed. It
// reaches the books and papers as well as the notes, so it asks about the vault
// and not about a note.
func addNoteSearch(server *sdk.Server, core Core) {
	sdk.AddTool(server, &sdk.Tool{
		Name:  "note_search",
		Title: "Search the vault",
		Description: "Search everything the vault holds — the notes, and the books and " +
			"papers filed beside them — for the words typed and for what they mean. " +
			"Returns passages, best first: the text around each hit and the file it was " +
			"read from. A file answers with at most a few of its passages, so a long " +
			"book does not take the answer. " +
			"A passage is a window cut to a size, and it ends where it was cut, which " +
			"is mid-sentence as often as not: read on with source_read before " +
			"concluding that a book says nothing about something. " +
			"Use this before assuming something is or is not written down. " +
			"When the person asks about a book — find it in the book, what does the " +
			"book say — pass kinds: [\"book\"]. A vault holds far more notes than " +
			"books, and a search told to look everywhere answers with notes.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		Query string   `json:"query" jsonschema:"words to look for"`
		Kinds []string `json:"kinds,omitempty" jsonschema:"which sorts of file to look in: note, book. All of them when left out"`
		Limit int      `json:"limit,omitempty" jsonschema:"how many passages to return, 20 by default"`
	}) (*sdk.CallToolResult, struct {
		Matches []Passage `json:"matches"`
	}, error) {
		type out = struct {
			Matches []Passage `json:"matches"`
		}
		if in.Limit > maxMatches {
			return nil, out{}, fmt.Errorf("ask for at most %d passages at a time", maxMatches)
		}
		of, err := sorts(in.Kinds)
		if err != nil {
			return nil, out{}, err
		}
		found, err := core.Notes.Search.Execute(ctx, core.shown().Vault, in.Query,
			search.Parameters{Kinds: of, Limit: in.Limit, Each: passagesEach})
		if err != nil {
			return nil, out{}, err
		}
		matches := make([]Passage, 0, len(found))
		for _, p := range found {
			matches = append(matches, Passage{
				Source:   p.Source,
				Location: p.Location,
				Text:     p.Text,
				Start:    p.Start,
				Length:   p.Length,
			})
		}
		return nil, out{Matches: matches}, nil
	})
}

// sorts is the kinds of source a question names. A search reaches notes and
// books, and a kind outside those two is refused and said so.
func sorts(named []string) ([]domain.SourceKind, error) {
	out := make([]domain.SourceKind, 0, len(named))
	for _, one := range named {
		kind := domain.SourceKind(one)
		if kind != domain.KindNote && kind != domain.KindBook {
			return nil, fmt.Errorf("%q is not a sort of file a search reaches: try %q or %q",
				one, domain.KindNote, domain.KindBook)
		}
		out = append(out, kind)
	}
	return out, nil
}
