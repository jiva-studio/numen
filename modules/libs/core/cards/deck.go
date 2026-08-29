package cards

import (
	"bytes"
	"regexp"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/markdown"
)

// ReadDeck reads the body of a note as a deck. Which notes are decks is what
// the frontmatter key `type` says, and this reads whichever it is handed.
//
// It never fails: a deck that could not be understood is still somebody's
// writing, and every value in it is read whatever else is wrong.
func ReadDeck(n domain.Note) Deck {
	deck, _ := readDeck(n.Ref, []byte(n.Body))
	return deck
}

// loneLinkRe is a paragraph that is one wikilink and nothing else. Under a
// card's heading that names the stencil the card is cut by.
var loneLinkRe = regexp.MustCompile(`^\[\[([^\]\[]+)\]\]$`)

// headSpan is where one heading line stands: the byte it begins at, and the
// byte after it.
type headSpan struct {
	head int
	from int
}

// sectionHeads is where each of a deck's sections opens, in the order they
// stand in the file.
func sectionHeads(body []byte) []headSpan {
	var out []headSpan
	for _, s := range sections(body, SectionLevel, FieldLevel) {
		if s.level == SectionLevel {
			out = append(out, headSpan{head: s.head, from: s.from})
		}
	}
	return out
}

// opens reports whether a heading of this level begins something of the deck's
// own, which is what a section's text and a card's values stop at.
func opens(level int) bool { return level == SectionLevel || level == CardLevel }

// cardSpan is where one card and each of its values stand in the body.
type cardSpan struct {
	name string
	mark string
	// head is the byte the heading line begins at and from is the byte after
	// it, which is the run the first field's value occupies. end is where the
	// card stops: the next card, or the end of the file.
	head   int
	from   int
	end    int
	values []valueSpan
}

// valueSpan is where one field of a card stands: its heading line, and the
// value under it.
type valueSpan struct {
	field    string
	head     int
	from, to int
}

func readDeck(ref domain.FileRef, body []byte) (Deck, []cardSpan) {
	d := Deck{Ref: ref}
	secs := sections(body, SectionLevel, FieldLevel)

	first := len(body)
	for _, s := range secs {
		if opens(s.level) {
			first = s.head
			break
		}
	}
	d.Preamble = markdown.Normalised(string(body[:first]))

	read := 0
	under := NoSection
	var spans []cardSpan
	for i, s := range secs {
		if !opens(s.level) {
			continue
		}

		end := len(body)
		for _, later := range secs[i+1:] {
			if opens(later.level) {
				end = later.head
				break
			}
		}

		if s.level == SectionLevel {
			// Everything down to the section's first card is the person's own
			// writing about their deck, whatever it is made of.
			lead, leadEnd := run(body, s.from, end)
			d.Sections = append(d.Sections, Section{Name: s.name, Lead: lead})
			under = len(d.Sections) - 1
			read = trimmedEnd(body, s.head, s.from)
			if lead != "" {
				read = leadEnd
			}
			continue
		}

		at := len(d.Cards)
		heading, carried := ReadHeading(s.name)
		card := Card{Heading: heading, Mark: carried, Section: under}
		span := cardSpan{name: s.name, mark: carried, head: s.head, from: s.from, end: end}

		target, leadFrom := stencil(body, s.from, s.to)
		card.Stencil = target
		read = trimmedEnd(body, s.head, s.from)
		if target != "" {
			read = trimmedEnd(body, s.from, leadFrom)
		}
		lead, leadEnd := run(body, leadFrom, s.to)
		card.Lead = lead
		if lead != "" {
			read = leadEnd
		}
		if target == "" {
			d.Problems = append(d.Problems, against(at, CheckNoStencil,
				"the first paragraph of this card is not a lone wikilink, so it names no stencil"))
		}

		fields := map[string]bool{}
		for _, f := range secs[i+1:] {
			if f.level != FieldLevel {
				break
			}
			value, valueEnd := run(body, f.from, f.to)
			read = trimmedEnd(body, f.head, f.from)
			if value != "" {
				read = valueEnd
			}
			card.Values = append(card.Values, Value{Field: f.name, Text: value})
			span.values = append(span.values, valueSpan{field: f.name, head: f.head, from: f.from, to: f.to})
			if fields[f.name] {
				problem := against(at, CheckTwoValues, "this card writes "+f.name+" twice")
				problem.Field = f.name
				d.Problems = append(d.Problems, problem)
			}
			fields[f.name] = true
		}

		d.Cards = append(d.Cards, card)
		spans = append(spans, span)
	}

	d.Problems = append(d.Problems, twoMarks(d.Cards)...)
	if len(d.Cards) > 0 || len(d.Sections) > 0 {
		d.Tail = markdown.Normalised(string(body[read:]))
	}
	return d, spans
}

// twoMarks is one problem against each card of a mark another card in this deck
// carries. Both are read and both are shown marked: nothing a person wrote goes
// missing from the screen, and which of the two is meant is a thing only they
// know. Neither is given another mark, because choosing would be choosing which
// of the two keeps its history.
func twoMarks(cs []Card) []Problem {
	carried := map[string]int{}
	for _, c := range cs {
		if c.Mark != "" {
			carried[c.Mark]++
		}
	}
	var out []Problem
	for at, c := range cs {
		if c.Mark != "" && carried[c.Mark] > 1 {
			out = append(out, against(at, CheckTwoMarks,
				"another card in this deck carries the mark "+c.Mark))
		}
	}
	return out
}

// stencil reads the lone wikilink paragraph directly beneath a card's heading.
// It returns what the link says and the byte the lead begins at; a card whose
// first paragraph is something else names no stencil, and the whole of what is
// under the heading is lead.
func stencil(body []byte, from, to int) (target string, leadFrom int) {
	at := from
	for at < to {
		end, next := to, to
		if i := bytes.IndexByte(body[at:to], '\n'); i >= 0 {
			end, next = at+i, at+i+1
		}
		line := strings.TrimSpace(strings.TrimRight(string(body[at:end]), "\r"))
		if line == "" {
			at = next
			continue
		}
		if m := loneLinkRe.FindStringSubmatch(line); m != nil && paragraphEnds(body, next, to) {
			return m[1], next
		}
		return "", from
	}
	return "", from
}

// paragraphEnds reports whether the line just read stood on its own: the next
// line is blank, or there is no next line. A wikilink with prose under it is a
// paragraph that begins with a link.
func paragraphEnds(body []byte, at, to int) bool {
	if at >= to {
		return true
	}
	end := to
	if i := bytes.IndexByte(body[at:to], '\n'); i >= 0 {
		end = at + i
	}
	return strings.TrimSpace(string(body[at:end])) == ""
}
