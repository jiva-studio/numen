package format

import (
	"bytes"
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/markdown"
)

// ReadDeck reads the body of a note as a deck, whichever note it is given: the frontmatter key
// `type` decides which notes are decks, not this function. It never fails: a deck that cannot be
// fully interpreted still has its content, and every value in it is read regardless of other
// problems.
func ReadDeck(n domain.Note) Deck {
	deck, _ := readDeck(n.FileStat, []byte(n.Body))
	return deck
}

// headSpan is the position of one heading line: the byte it begins at, and the
// byte after it.
type headSpan struct {
	head int
	from int
}

func sectionHeads(body []byte) []headSpan {
	var out []headSpan
	for _, s := range sections(body, SectionLevel, FieldLevel) {
		if s.level == SectionLevel {
			out = append(out, headSpan{head: s.head, from: s.from})
		}
	}
	return out
}

func isDeckHeadingLevel(level int) bool { return level == SectionLevel || level == CardLevel }

// cardSpan is where one card and each of its values stand in the body.
type cardSpan struct {
	name     string
	cardID   ID
	head     int
	from     int
	end      int
	linkFrom int
	linkTo   int
	values   []valueSpan
}

// valueSpan is the position of one field of a card: its heading line, and the
// value after it.
type valueSpan struct {
	field    string
	head     int
	from, to int
}

type deckBuilder struct {
	deck           Deck
	spans          []cardSpan
	read           int
	currentSection int
}

func findNextDeckHeading(secs []section, start int, bodyLen int) int {
	for _, later := range secs[start:] {
		if isDeckHeadingLevel(later.level) {
			return later.head
		}
	}
	return bodyLen
}

func (b *deckBuilder) addSection(s section, body []byte, end int) {
	lead, leadEnd := readTrimmedRun(body, s.from, end)
	b.deck.Sections = append(b.deck.Sections, Section{Name: s.name, Preamble: lead})
	b.currentSection = len(b.deck.Sections) - 1
	b.read = getTrimmedEnd(body, s.head, s.from)
	if lead != "" {
		b.read = leadEnd
	}
}

func (b *deckBuilder) readCardFields(body []byte, secs []section, pos int, card *Card, span *cardSpan) {
	fields := map[string]bool{}
	for _, f := range secs {
		if f.level != FieldLevel {
			break
		}
		value, valueEnd := readTrimmedRun(body, f.from, f.to)
		b.read = getTrimmedEnd(body, f.head, f.from)
		if value != "" {
			b.read = valueEnd
		}
		card.Values = append(card.Values, Value{Field: f.name, Text: value})
		span.values = append(span.values, valueSpan{field: f.name, head: f.head, from: f.from, to: f.to})
		if fields[f.name] {
			problem := newCardProblem(pos, FaultTwoValues, "this card writes "+f.name+" twice")
			problem.Field = f.name
			b.deck.Problems = append(b.deck.Problems, problem)
		}
		fields[f.name] = true
	}
}

func (b *deckBuilder) addCard(body []byte, s section, secsAfter []section, end int) {
	pos := len(b.deck.Cards)
	heading, carried := ReadHeading(s.name)
	card := Card{
		Heading: heading, CardID: carried, CardModelLink: "", Preamble: "",
		Section: b.currentSection, Values: nil,
	}
	span := cardSpan{
		name: s.name, cardID: carried, head: s.head, from: s.from, end: end,
		linkFrom: s.from, linkTo: s.from, values: nil,
	}

	target, leadFrom := readCardModelLink(body, s.from, s.to)
	card.CardModelLink = target
	b.read = getTrimmedEnd(body, s.head, s.from)
	if target != "" {
		b.read = getTrimmedEnd(body, s.from, leadFrom)
		span.linkFrom, span.linkTo = lineFrom(body, s.from, b.read), b.read
	}
	lead, leadEnd := readTrimmedRun(body, leadFrom, s.to)
	card.Preamble = lead
	if lead != "" {
		b.read = leadEnd
	}
	if target == "" {
		b.deck.Problems = append(b.deck.Problems, newCardProblem(pos, FaultNoCardModel,
			"the first paragraph of this card is not a lone wikilink, so it names no card model"))
	}

	b.readCardFields(body, secsAfter, pos, &card, &span)

	b.deck.Cards = append(b.deck.Cards, card)
	b.spans = append(b.spans, span)
}

func createEmptyStat() domain.FileStat {
	// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="Empty FileStat Size is unused by internal span readers."
	return domain.FileStat{Path: "", Kind: "", Size: 0, ModTime: time.Time{}}
}

func readDeck(ref domain.FileStat, body []byte) (Deck, []cardSpan) {
	secs := sections(body, SectionLevel, FieldLevel)
	b := deckBuilder{
		deck: Deck{
			Ref: ref, Preamble: "", Tail: "", Sections: nil, Cards: nil, Problems: nil,
		},
		spans: nil,
		// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="Initial read offset is only used when Cards or Sections is non-empty, which overwrites read."
		read:           0,
		currentSection: NoSection,
	}

	first := len(body)
	for _, s := range secs {
		if isDeckHeadingLevel(s.level) {
			first = s.head
			break
		}
	}
	b.deck.Preamble = markdown.Normalize(string(body[:first]))

	for i, s := range secs {
		if !isDeckHeadingLevel(s.level) {
			continue
		}
		end := findNextDeckHeading(secs, i+1, len(body))
		if s.level == SectionLevel {
			b.addSection(s, body, end)
		} else {
			b.addCard(body, s, secs[i+1:], end)
		}
	}

	b.deck.Problems = append(b.deck.Problems, findDuplicateCardIDs(b.deck.Cards)...)
	if len(b.deck.Cards) > 0 || len(b.deck.Sections) > 0 {
		b.deck.Tail = markdown.Normalize(string(body[b.read:]))
	}
	return b.deck, b.spans
}

func findDuplicateCardIDs(cs []Card) []Problem {
	carried := map[ID]int{}
	for _, c := range cs {
		if c.CardID != "" {
			carried[c.CardID]++
		}
	}
	var out []Problem
	for item, c := range cs {
		// gomutants:disable-next-line EXPRESSION_REMOVE reason="carried[\"\"] is always 0, so carried[c.CardID] > 1 already excludes empty CardID."
		if c.CardID != "" && carried[c.CardID] > 1 {
			out = append(out, newCardProblem(item, FaultDuplicateCardID,
				"another card in this deck has the card id "+string(c.CardID)))
		}
	}
	return out
}

func readCardModelLink(body []byte, from, to int) (target string, leadFrom int) {
	pos := from
	for pos < to {
		end, next := to, to
		if i := bytes.IndexByte(body[pos:to], '\n'); i >= 0 {
			end, next = pos+i, pos+i+1
		}
		line := strings.TrimSpace(strings.TrimRight(string(body[pos:end]), "\r"))
		if line == "" {
			pos = next
			continue
		}
		// A paragraph that is one wikilink and nothing else names the card model.
		found := markdown.WikilinksIn(line)
		if len(found) == 1 && found[0].At == 0 && found[0].To == len(line) &&
			paragraphEnds(body, next, to) {
			return found[0].Inside, next
		}
		return "", from
	}
	return "", from
}

func paragraphEnds(body []byte, pos, to int) bool {
	// gomutants:disable-next-line CONDITIONALS_BOUNDARY,BRANCH_IF,RETURN_FALSE reason="When pos >= to, body[pos:to] is empty and trims to empty string, returning true below."
	if pos >= to {
		return true
	}
	end := to
	// gomutants:disable-next-line INTEGER_DECREMENT reason="bytes.IndexByte returns -1 when not found and >= 0 when found."
	if i := bytes.IndexByte(body[pos:to], '\n'); i >= 0 {
		end = pos + i
	}
	return strings.TrimSpace(string(body[pos:end])) == ""
}
