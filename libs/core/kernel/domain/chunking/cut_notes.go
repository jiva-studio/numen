package chunking

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// CutNotes cuts notes into the headings and chunks the index keeps. A zero
// Sizes and Legibility take the chunking defaults.
type CutNotes struct {
	Sizes      Sizes
	Legibility Legibility
	Profiles   func(domain.NoteType) (domain.NoteProfile, bool)
}

func (u CutNotes) getProfile(noteType domain.NoteType) domain.NoteProfile {
	if u.Profiles != nil {
		if profile, declared := u.Profiles(noteType); declared {
			return profile
		}
	}
	return domain.DefaultProfile()
}

// NewCutNotes cuts at the bound a model reading maxTokens tokens allows, and at
// the chunking defaults where maxTokens is zero.
func NewCutNotes(maxTokens int) CutNotes {
	return CutNotes{Sizes: Sizes{Limit: GetCharacterBound(maxTokens)}}
}

// noteCutReader names the algorithm that cuts a note. It changes with the algorithm, so that notes cut
// by the old one are cut again.
const noteCutReader = "note-cut-1"

// Recipe names how this cut turns a note into chunks: the sizes it cuts at.
func (u CutNotes) Recipe() string {
	sizes := u.getSizes().Resolve()
	return fmt.Sprintf("%s/large=%d+%d/small=%d+%d/limit=%d",
		noteCutReader, sizes.Large, sizes.LargeOverlap, sizes.Small, sizes.SmallOverlap, sizes.Limit)
}

func (u CutNotes) getSizes() Sizes {
	sizes := u.Sizes
	sizes.Large = Whole
	return sizes
}

// Execute gives each note with its indexed headings and the chunks of its body.
func (u CutNotes) Execute(notes []domain.Note) []domain.NoteChunks {
	out := make([]domain.NoteChunks, 0, len(notes))
	recipe := u.Recipe()
	for _, n := range notes {
		profile := u.getProfile(n.Type)
		headings := getIndexedHeadings(n, profile)
		out = append(out, domain.NoteChunks{Note: n, Headings: headings, Chunks: u.cut(n, headings, profile), Recipe: recipe})
	}
	return out
}

func (u CutNotes) cut(n domain.Note, headings []domain.Heading, profile domain.NoteProfile) []domain.Chunk {
	if !profile.HasChunks {
		return nil
	}

	pos := max(int(n.FileStat.Size)-len(n.Body), 0)
	sizes := u.getSizes()
	body := n.Body

	out := Chunks(body, getPartStarts(headings), sizes, u.Legibility,
		func(c Chunk, isLarge bool) domain.Chunk {
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

func getIndexedHeadings(n domain.Note, profile domain.NoteProfile) []domain.Heading {
	if profile.HeadingLevels == nil && len(profile.IDHeadingLevels) == 0 {
		return n.Headings
	}
	out := make([]domain.Heading, 0, len(n.Headings))
	for _, h := range n.Headings {
		if profile.HeadingLevels != nil && !slices.Contains(profile.HeadingLevels, h.Level) {
			continue
		}
		if slices.Contains(profile.IDHeadingLevels, h.Level) {
			h.Text = removeID(h.Text, profile.IDPattern)
			if h.Text == "" {
				continue
			}
		}
		out = append(out, h)
	}
	return out
}

func removeID(heading string, pattern *regexp.Regexp) string {
	pos := strings.LastIndex(heading, " ")
	last := heading
	if pos >= 0 {
		last = heading[pos+1:]
	}
	word, found := strings.CutPrefix(last, "^")
	if !found || word == "" || !isIDWord(word, pattern) {
		return heading
	}
	if pos < 0 {
		return ""
	}
	return heading[:pos]
}

func isIDWord(word string, pattern *regexp.Regexp) bool {
	if pattern != nil {
		return pattern.MatchString(word)
	}
	return !strings.ContainsFunc(word, func(r rune) bool {
		return !(r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r == '-')
	})
}
