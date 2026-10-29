package cardtools

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/api"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/edit"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/cards/deckedit"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/format"
)

// How much one call may ask about the cards of a vault. A deck is a file holding what would
// otherwise be a folder of notes, so a deck with no ceiling on it is a way to put a vault in a
// context window by accident.
const (
	maxCardModels = 50
	maxCards      = 50
	// maxBytes is how much one call may put in a deck.
	maxBytes = edit.MaxBytes
)

func stamp(t time.Time) int64 {
	return t.UnixNano()
}

func instant(nanos int64) time.Time {
	return time.Unix(0, nanos)
}

// CardModel is a card model as the list of them names it.
type CardModel struct {
	Path string `json:"path" jsonschema:"the card model's path relative to the vault folder"`
	// Name is what a card's wikilink writes. A link is resolved by path and by filename, so the name
	// that reaches this card model is its filename without the extension.
	Name   string   `json:"name" jsonschema:"the name to write in a card's wikilink, which is the filename without its extension"`
	Title  string   `json:"title" jsonschema:"what the card model is called, which is what a person sees"`
	Fields []string `json:"fields,omitempty" jsonschema:"what a card cut by this card model is asked for, in order; absent for a card model that could not be read"`
}

// Card is one card as a deck answers with it.
type Card struct {
	CardID    string       `json:"cardId" jsonschema:"what the card is for as long as it exists, and how every tool here addresses it; empty for a card typed in by hand, which is given one the next time the deck is written"`
	Section   int          `json:"section" jsonschema:"where the section this card stands under stands in the deck's sections, counted from the first; -1 for a card standing before the first section"`
	CardModel string       `json:"cardModel,omitempty" jsonschema:"the card model this card is cut by, as the wikilink beneath its heading names it"`
	Values    []FieldValue `json:"values,omitempty" jsonschema:"what the card holds, in the order it stands in the file"`
}

// FieldValue is what somebody wrote under one of a card's fields.
type FieldValue struct {
	Field string `json:"field" jsonschema:"the field's name, spelled as the card model declares it"`
	Text  string `json:"text" jsonschema:"the HTML under that heading"`
}

// Fault is one thing wrong with a deck, on the card it is against.
type Fault struct {
	Card int    `json:"card" jsonschema:"where the card stands in the deck, counted from its first card; -1 for a fault against no card"`
	Why  string `json:"why" jsonschema:"what is wrong, and what the application did about it"`
}

// CurrentCard is the card in front of the person, as an agent is told about it. The deck is the file
type CurrentCard struct {
	Deck     string `json:"deck" jsonschema:"the deck the card stands in, by the path the vault files it under; empty when no card is in front of them"`
	Card     string `json:"card,omitempty" jsonschema:"the card's id, as card_read gives it"`
	Template string `json:"template,omitempty" jsonschema:"the template it is being shown through, spelled as the card model writes it"`
}

// CardTemplate is one way a card model shows a card.
type CardTemplate struct {
	Name  string `json:"name" jsonschema:"what the template is called"`
	Front string `json:"front" jsonschema:"what is shown before the answer, as HTML with {{Field}} where a value goes"`
	Back  string `json:"back" jsonschema:"what is shown after it"`
}

// WriteOutcome is what a deck became: where it is, and what to present at the next write of it.
type WriteOutcome struct {
	Path        string `json:"path"`
	Fingerprint string `json:"fingerprint" jsonschema:"hand this to the next write of this deck without reading it back"`
	Cards       int    `json:"cards" jsonschema:"how many cards the deck now holds"`
	CardID      string `json:"cardId,omitempty" jsonschema:"the card id the card just written is addressed by, for as long as it exists"`
}

func newWriteOutcome(path string, edited deckedit.EditResult) WriteOutcome {
	return WriteOutcome{Path: path, Fingerprint: fingerprintOf(edited.FileStat), Cards: edited.Cards}
}

func newCard(card format.Card) Card {
	out := Card{CardID: string(card.CardID), Section: card.Section, CardModel: card.CardModelLink}
	for _, v := range card.Values {
		out.Values = append(out.Values, FieldValue{Field: v.Field, Text: v.Text})
	}
	return out
}

func values(vs []FieldValue) []format.Value {
	out := make([]format.Value, 0, len(vs))
	for _, v := range vs {
		out = append(out, format.Value{Field: v.Field, Text: v.Text})
	}
	return out
}

func newCardTemplates(fs []CardTemplate) []format.CardTemplate {
	out := make([]format.CardTemplate, 0, len(fs))
	for _, f := range fs {
		out = append(out, format.CardTemplate{Name: f.Name, Front: f.Front, Back: f.Back})
	}
	return out
}

func newTemplates(templates []format.CardTemplate) []CardTemplate {
	out := make([]CardTemplate, 0, len(templates))
	for _, t := range templates {
		out = append(out, CardTemplate{Name: t.Name, Front: t.Front, Back: t.Back})
	}
	return out
}

// faults is what was wrong with a deck, on the card each stands against. The position is counted from
// the deck's first card, whichever run of them a call asked for.
func faults(problems []format.Problem) []Fault {
	if len(problems) == 0 {
		return nil
	}
	out := make([]Fault, 0, len(problems))
	for _, p := range problems {
		out = append(out, Fault{Card: p.Card, Why: p.Detail})
	}
	return out
}

// countBytes is how many bytes a call will put in a deck. A field's name lands in the file beside its
// value, so both are measured.
func countBytes(vs []FieldValue) int {
	size := 0
	for _, v := range vs {
		size += len(v.Field) + len(v.Text)
	}
	return size
}

// fingerprintOf is what a deck or a card model was when it was read, as an agent hands it back.
func fingerprintOf(ref domain.FileStat) string {
	return strconv.FormatInt(ref.Size, 10) + "-" + strconv.FormatInt(stamp(ref.ModTime), 10)
}

// parseFingerprint is the file stat a caller presents. A tool that changes what somebody may have
func parseFingerprint(s string) (domain.FileStat, error) {
	if s == "" {
		return domain.FileStat{}, errors.New("present the fingerprint the read gave you: card_read for a deck, card_model_read for a card model")
	}
	size, mtime, found := strings.Cut(s, "-")
	if !found {
		return domain.FileStat{}, fmt.Errorf("%q is not a fingerprint a read gave out", s)
	}
	ref := domain.FileStat{}
	var err error
	if ref.Size, err = strconv.ParseInt(size, 10, 64); err != nil {
		return domain.FileStat{}, fmt.Errorf("%q is not a fingerprint a read gave out", s)
	}
	nanos, err := strconv.ParseInt(mtime, 10, 64)
	if err != nil {
		return domain.FileStat{}, fmt.Errorf("%q is not a fingerprint a read gave out", s)
	}
	ref.ModTime = instant(nanos)
	return ref, nil
}

func convertOutcomeToErrorCode(o edit.ReadOutcome) (api.ErrorCode, bool) {
	switch o {
	case edit.Missing:
		return api.ErrorCodeMissing, true
	case edit.NotANote:
		return api.ErrorCodeNotANote, true
	case edit.NotText:
		return api.ErrorCodeNotText, true
	case edit.TooLarge:
		return api.ErrorCodeTooLarge, true
	case edit.Unreadable:
		return api.ErrorCodeUnreadable, true
	case edit.Ok:
		return api.ErrorCodeUnspecified, false
	}
	return api.ErrorCodeUnspecified, false
}

func describeCode(reason api.ErrorCode) string {
	switch reason {
	case api.ErrorCodeMissing:
		return "the vault holds no note at this path"
	case api.ErrorCodeNotANote:
		return "this is not a note the vault holds"
	case api.ErrorCodeNotText:
		return "this file is not text: some of it is not valid UTF-8, so open it as a file"
	case api.ErrorCodeTooLarge:
		return fmt.Sprintf("it is larger than the %d bytes this reads; open the file instead", edit.MaxBytes)
	case api.ErrorCodeBodyUnwritable:
		return "this text opens with a frontmatter delimiter, so it cannot be written into a note"
	case api.ErrorCodeUnreadable:
		return "the frontmatter of this note cannot be read, so it can be neither read nor written from here"
	case api.ErrorCodeAlreadyExists:
		return "a file is already where this note would go"
	case api.ErrorCodeInvalidName:
		return "a note cannot be called this"
	case api.ErrorCodeNotACardModel:
		return "this note is not a card model"
	case api.ErrorCodeNotADeck:
		return "this note is not a deck"
	case api.ErrorCodeDeckTooLarge:
		return "this deck is larger than a deck is read at"
	case api.ErrorCodeNotAPreset:
		return "this note is not a preset"
	case api.ErrorCodeStale:
		return "this file is no longer the one that was read, and nothing was written: read it again and write from what is there now"
	default:
		return string(reason)
	}
}
