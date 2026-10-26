package query

import (
	"regexp"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/core/internal/chunking"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// CutNotes cuts notes into the headings and chunks the index keeps. A zero
// Sizes and Legibility take the chunking defaults.
type CutNotes struct {
	Sizes      chunking.Sizes
	Legibility chunking.Legibility
	// Profiles says how the notes of a type a plugin declares are kept. Nil keeps every note as an
	// ordinary note is kept.
	Profiles func(domain.NoteType) (domain.NoteProfile, bool)
}

// getProfile is how the notes of the type are kept.
func (u CutNotes) getProfile(noteType domain.NoteType) domain.NoteProfile {
	if u.Profiles != nil {
		if profile, declared := u.Profiles(noteType); declared {
			return profile
		}
	}
	return domain.OrdinaryProfile()
}

// NewCutNotes cuts at the bound a model reading maxTokens tokens allows, and at
// the chunking defaults where maxTokens is zero.
func NewCutNotes(maxTokens int) CutNotes {
	return CutNotes{Sizes: chunking.Sizes{Limit: chunking.GetCharacterBound(maxTokens)}}
}

// Execute gives each note with its indexed headings and the chunks of its body.
func (u CutNotes) Execute(notes []domain.Note) []domain.NoteChunks {
	out := make([]domain.NoteChunks, 0, len(notes))
	for _, n := range notes {
		profile := u.getProfile(n.Type)
		headings := getIndexedHeadings(n, profile)
		out = append(out, domain.NoteChunks{Note: n, Headings: headings, Chunks: u.cut(n, headings, profile)})
	}
	return out
}

// cut is one large chunk over the whole note, holding the small chunks that
// carry the vectors. Each section the headings name is tiled on its own, and
// offsets are into the file, so the body's start after the frontmatter is added
// to every chunk. A note whose type has no chunks is found by its headings and its
// title.
func (u CutNotes) cut(n domain.Note, headings []domain.Heading, profile domain.NoteProfile) []domain.Chunk {
	if !profile.HasChunks {
		return nil
	}

	pos := max(int(n.Fingerprint.Size)-len(n.Body), 0)
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
			return domain.Chunk{Start: pos + c.Start, Length: c.Length, Location: c.Location, Text: text}
		})
	if len(out) == 0 {
		// A note of a title and no words is answered by its title.
		out = append(out, domain.Chunk{Start: pos, Length: len(body), Text: n.Title})
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

// getIndexedHeadings is the headings of a note as the index keeps them.
//
// A type a plugin declares keeps the levels of heading its profile names, and a heading of a level
// that ends in a mark is kept without it; a heading of no text is not kept. Every other note keeps
// every heading it has.
func getIndexedHeadings(n domain.Note, profile domain.NoteProfile) []domain.Heading {
	if profile.HeadingLevels == nil && len(profile.MarkedLevels) == 0 {
		return n.Headings
	}
	out := make([]domain.Heading, 0, len(n.Headings))
	for _, h := range n.Headings {
		if profile.HeadingLevels != nil && !slices.Contains(profile.HeadingLevels, h.Level) {
			continue
		}
		if slices.Contains(profile.MarkedLevels, h.Level) {
			h.Text = withoutMark(h.Text, profile.Mark)
			if h.Text == "" {
				continue
			}
		}
		out = append(out, h)
	}
	return out
}

// withoutMark is a heading without the mark it ends in: a space, a caret and a word of letters,
// digits and hyphens. A heading that is only a mark has no text.
func withoutMark(heading string, mark *regexp.Regexp) string {
	pos := strings.LastIndex(heading, " ")
	last := heading
	if pos >= 0 {
		last = heading[pos+1:]
	}
	word, found := strings.CutPrefix(last, "^")
	if !found || word == "" || !isMarkWord(word, mark) {
		return heading
	}
	if pos < 0 {
		return ""
	}
	return heading[:pos]
}

// isMarkWord says whether a word is the word of a mark: one the pattern matches whole, else any word
// of letters, digits and hyphens.
func isMarkWord(word string, mark *regexp.Regexp) bool {
	if mark != nil {
		return mark.MatchString(word)
	}
	return !strings.ContainsFunc(word, func(r rune) bool {
		return !(r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r == '-')
	})
}
