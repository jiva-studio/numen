package cardtools

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/api"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/edit"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/format"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/cards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/cards/deckedit"
)

// Limits on how much one call may request about the cards of a vault. A deck is a file holding what would
// be a folder of notes, so an unbounded deck read can fill a context window.
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
	Path   string   `json:"path" jsonschema:"the card model's path relative to the vault folder"`
	Name   string   `json:"name" jsonschema:"the name to write in a card's wikilink, which is the filename without its extension"`
	Title  string   `json:"title" jsonschema:"the title of the card model"`
	Fields []string `json:"fields,omitempty" jsonschema:"the field names of the card model, in order; absent for a card model that could not be read"`
}

// Card is one card as a deck answers with it.
type Card struct {
	CardID       string       `json:"cardId" jsonschema:"the stable identifier of the card, which every card tool takes; empty for a card written by hand, which gets one at the next write of the deck"`
	SectionIndex int          `json:"sectionIndex" jsonschema:"the index of the section of the card in the deck, counted from 0; -1 for a card before the first section"`
	CardModel    string       `json:"cardModel,omitempty" jsonschema:"the card model of the card, as named by the wikilink under its heading"`
	Values       []FieldValue `json:"values,omitempty" jsonschema:"the field values, in file order"`
}

// FieldValue is the text of one of a card's fields.
type FieldValue struct {
	Field string `json:"field" jsonschema:"the field's name, spelled as the card model declares it"`
	Text  string `json:"text" jsonschema:"the HTML of the field value"`
}

// Problem is one problem of a deck, on the card it concerns.
type Problem struct {
	CardIndex int    `json:"cardIndex" jsonschema:"the index of the card in the deck, counted from 0; -1 for a problem of no card"`
	Message   string `json:"message" jsonschema:"the problem and the action the application took"`
}

// CurrentCard is the card the user is viewing, as reported to an agent. The path is the deck file containing it.
type CurrentCard struct {
	Path         string `json:"path" jsonschema:"the path of the deck of the card, relative to the vault folder; empty when no card is open"`
	CardID       string `json:"cardId,omitempty" jsonschema:"the id of the card, as card_read returns it"`
	TemplateName string `json:"templateName,omitempty" jsonschema:"the name of the template the card is shown with, as the card model spells it"`
}

// CardTemplate is one way a card model shows a card.
type CardTemplate struct {
	Name  string `json:"name" jsonschema:"the name of the template"`
	Front string `json:"front" jsonschema:"the front side, as HTML with {{Field}} placeholders for the values"`
	Back  string `json:"back" jsonschema:"the back side, as HTML with {{Field}} placeholders for the values"`
}

// WriteOutcome is what a deck became: where it is, and what to present at the next write of it.
type WriteOutcome struct {
	Path        string `json:"path" jsonschema:"the path of the deck, relative to the vault folder"`
	Fingerprint string `json:"fingerprint" jsonschema:"the new deck fingerprint; pass it to the next write of this deck"`
	Total       int    `json:"total" jsonschema:"the number of cards in the deck"`
	CardID      string `json:"cardId,omitempty" jsonschema:"the identifier of the written card"`
}

func newWriteOutcome(path string, edited deckedit.EditResult) WriteOutcome {
	return WriteOutcome{Path: path, Fingerprint: fingerprintOf(edited.FileStat), Total: edited.Cards, CardID: ""}
}

func newCard(card format.Card) Card {
	out := Card{CardID: string(card.CardID), SectionIndex: card.Section, CardModel: card.CardModelLink, Values: nil}
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
		out = append(out, format.CardTemplate{Name: f.Name, Preamble: "", Front: f.Front, Back: f.Back})
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

func getProblems(problems []format.Problem) []Problem {
	// gomutants:disable-next-line INTEGER_DECREMENT,BRANCH_IF reason="Slice length is never negative, and an empty slice with omitempty serializes identically to nil."
	if len(problems) == 0 {
		return nil
	}
	out := make([]Problem, 0, len(problems))
	for _, p := range problems {
		out = append(out, Problem{CardIndex: p.Card, Message: p.Detail})
	}
	return out
}

func countBytes(vs []FieldValue) int {
	size := 0
	for _, v := range vs {
		size += len(v.Field) + len(v.Text)
	}
	return size
}

func fingerprintOf(ref domain.FileStat) string {
	return strconv.FormatInt(ref.Size, 10) + "-" + strconv.FormatInt(stamp(ref.ModTime), 10)
}

func parseFingerprint(s string) (domain.FileStat, error) {
	if s == "" {
		return domain.FileStat{}, errors.New("present the fingerprint the read gave you: card_read for a deck, card_model_read for a card model")
	}
	size, mtime, found := strings.Cut(s, "-")
	if !found {
		return domain.FileStat{}, fmt.Errorf("%q is not a fingerprint a read gave out", s)
	}
	ref := domain.FileStat{Path: "", Kind: "", Size: 0, ModTime: time.Time{}}
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
		return api.ErrorCodeNotFound, true
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
	descriptions := map[api.ErrorCode]string{
		api.ErrorCodeNotFound:       "the vault has no note at this path",
		api.ErrorCodeNotANote:       "this file is not a note",
		api.ErrorCodeNotText:        "this file is not text: some of it is not valid UTF-8, so open it as a file",
		api.ErrorCodeTooLarge:       fmt.Sprintf("it is larger than the %d bytes this reads; open the file instead", edit.MaxBytes),
		api.ErrorCodeBodyUnwritable: "this text opens with a frontmatter delimiter, so it cannot be written into a note",
		api.ErrorCodeUnreadable:     "the frontmatter of this note cannot be read, so it can be neither read nor written from here",
		api.ErrorCodeAlreadyExists:  "a file is already where this note would go",
		api.ErrorCodeInvalidName:    "a note cannot be called this",
		api.ErrorCodeNotACardModel:  "this note is not a card model",
		api.ErrorCodeNotADeck:       "this note is not a deck",
		api.ErrorCodeDeckTooLarge:   "this deck is larger than a deck is read at",
		api.ErrorCodeNotAPreset:     "this note is not a preset",
		api.ErrorCodeConflict:       "the file changed since it was read, and nothing was written: read it again and write from its current content",
	}
	if text, ok := descriptions[reason]; ok {
		return text
	}
	return string(reason)
}

// NoteCreated is the deck or the card model a create call created.
type NoteCreated struct {
	Path  string `json:"path" jsonschema:"the path of the new file, relative to the vault folder"`
	ID    string `json:"id" jsonschema:"the identifier of the new note"`
	Title string `json:"title" jsonschema:"the title of the new note"`
}

func newNoteCreated(created cards.CreateNoteResult, err error) (NoteCreated, error) {
	if err != nil {
		return NoteCreated{}, err
	}
	return NoteCreated{Path: created.Path, ID: created.ID, Title: created.Title}, nil
}
