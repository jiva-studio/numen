package cards

import (
	"bytes"
	"regexp"
	"strings"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/markdown"
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

// cardSpan is where one card and each of its values stand in the body.
type cardSpan struct {
	name string
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
	secs := sections(body)

	firstCard := len(body)
	for _, s := range secs {
		if s.level == 2 {
			firstCard = s.head
			break
		}
	}
	d.Preamble = markdown.Normalised(string(body[:firstCard]))

	named := map[string]bool{}
	read := 0
	var spans []cardSpan
	for i, s := range secs {
		if s.level != 2 {
			continue
		}
		at := len(d.Cards)

		end := len(body)
		for _, later := range secs[i+1:] {
			if later.level == 2 {
				end = later.head
				break
			}
		}

		card := Card{Name: s.name}
		span := cardSpan{name: s.name, head: s.head, from: s.from, end: end}

		switch {
		case s.name == "":
			d.Problems = append(d.Problems, against(at, CheckNoName, "this card has no name"))
		case named[s.name]:
			d.Problems = append(d.Problems, against(at, CheckTwoCards, "another card is called "+s.name))
		default:
			named[s.name] = true
		}

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
			if f.level != 3 {
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

	if len(d.Cards) > 0 {
		d.Tail = markdown.Normalised(string(body[read:]))
	}
	return d, spans
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
