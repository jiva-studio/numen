package note

import (
	"github.com/jiva-studio/numen/modules/libs/core/flashcards/format"
	"github.com/jiva-studio/numen/modules/libs/core/internal/chunking"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// CutNotes cuts notes into the headings and chunks the index keeps. A zero
// Sizes and Legibility take the chunking defaults.
type CutNotes struct {
	Sizes      chunking.Sizes
	Legibility chunking.Legibility
}

// NewCutNotes cuts at the bound a model reading maxTokens tokens allows when the
// installation embeds, and at the chunking defaults when it does not.
func NewCutNotes(isEmbedding bool, maxTokens int) CutNotes {
	if !isEmbedding {
		return CutNotes{}
	}
	return CutNotes{Sizes: chunking.Sizes{Limit: chunking.GetCharacterBound(maxTokens)}}
}

// Execute gives each note with its indexed headings and the chunks of its body.
func (u CutNotes) Execute(notes []domain.Note) []domain.NoteChunks {
	out := make([]domain.NoteChunks, 0, len(notes))
	for _, n := range notes {
		headings := getIndexedHeadings(n)
		out = append(out, domain.NoteChunks{Note: n, Headings: headings, Chunks: u.cut(n, headings)})
	}
	return out
}

// cut is one large chunk over the whole note, holding the small chunks that
// carry the vectors. Each section the headings name is tiled on its own, and
// offsets are into the file, so the body's start after the frontmatter is added
// to every chunk. A deck and a stencil have no chunks: a card is found by its
// heading and a stencil by its title.
func (u CutNotes) cut(n domain.Note, headings []domain.Heading) []domain.Chunk {
	if n.Type == domain.TypeDeck || n.Type == domain.TypeStencil {
		return nil
	}

	at := max(int(n.Fingerprint.Size)-len(n.Body), 0)
	sizes := u.Sizes
	sizes.Large = chunking.Whole
	body := n.Body

	out := chunking.Chunks(body, getPartStarts(headings), sizes, u.Legibility,
		func(c chunking.Chunk, isLarge bool) domain.Chunk {
			text := c.ExtractText(body)
			if isLarge {
				// The title is searched together with the body.
				text = n.Title + "\n" + text
			}
			return domain.Chunk{Start: at + c.Start, Length: c.Length, Location: c.Location, Text: text}
		})
	if len(out) == 0 {
		// A note of a title and no words is answered by its title.
		out = append(out, domain.Chunk{Start: at, Length: len(body), Text: n.Title})
	}
	return out
}

// getPartStarts is where each heading begins its section, named by its text.
func getPartStarts(headings []domain.Heading) []domain.PartStart {
	if len(headings) == 0 {
		return nil
	}
	out := make([]domain.PartStart, 0, len(headings))
	for _, h := range headings {
		out = append(out, domain.PartStart{Title: h.Text, Offset: h.Offset})
	}
	return out
}

// The two levels a deck spends on what a person writes: a section, and a card
// under it.
const (
	sectionLevel = 1
	cardLevel    = 2
)

// getIndexedHeadings is the headings of a note as the index keeps them.
//
// A deck keeps its sections and its cards, and a card's heading is kept without
// the mark it ends in; a heading of no text is not kept. A stencil keeps none.
// Every other note keeps every heading it has.
func getIndexedHeadings(n domain.Note) []domain.Heading {
	switch n.Type {
	case domain.TypeStencil:
		return nil
	case domain.TypeDeck:
		out := make([]domain.Heading, 0, len(n.Headings))
		for _, h := range n.Headings {
			switch h.Level {
			case sectionLevel:
				out = append(out, h)
			case cardLevel:
				h.Text, _ = format.ReadHeading(h.Text)
				if h.Text == "" {
					continue
				}
				out = append(out, h)
			}
		}
		return out
	default:
		return n.Headings
	}
}
