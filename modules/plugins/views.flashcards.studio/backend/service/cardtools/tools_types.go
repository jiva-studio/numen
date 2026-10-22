package cardtools

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	"github.com/jiva-studio/numen/modules/libs/notes/edit"
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/cards/deckedit"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/format"
)

// How much one call may ask about the cards of a vault. A deck is a file holding what would
// otherwise be a folder of notes, so a deck with no ceiling on it is a way to put a vault in a
// context window by accident.
const (
	maxStencils = 50
	maxCards    = 50
	// maxBytes is how much one call may put in a deck.
	maxBytes = edit.MaxBytes
)

func stamp(t time.Time) int64 {
	return t.UnixNano()
}

func instant(nanos int64) time.Time {
	return time.Unix(0, nanos)
}

// Stencil is a stencil as the list of them names it.
type Stencil struct {
	Path string `json:"path" jsonschema:"the stencil's path relative to the vault folder"`
	// Name is what a card's wikilink writes. A link is resolved by path and by filename, so the name
	// that reaches this stencil is its filename without the extension.
	Name   string   `json:"name" jsonschema:"the name to write in a card's wikilink, which is the filename without its extension"`
	Title  string   `json:"title" jsonschema:"what the stencil is called, which is what a person sees"`
	Fields []string `json:"fields,omitempty" jsonschema:"what a card cut by this stencil is asked for, in order; absent for a stencil that could not be read"`
}

// Card is one card as a deck answers with it.
type Card struct {
	Mark    string       `json:"mark" jsonschema:"what the card is for as long as it exists, and how every tool here addresses it; empty for a card typed in by hand, which is given one the next time the deck is written"`
	Section int          `json:"section" jsonschema:"where the section this card stands under stands in the deck's sections, counted from the first; -1 for a card standing before the first section"`
	Stencil string       `json:"stencil,omitempty" jsonschema:"the stencil this card is cut by, as the wikilink beneath its heading names it"`
	Values  []FieldValue `json:"values,omitempty" jsonschema:"what the card holds, in the order it stands in the file"`
}

// FieldValue is what somebody wrote under one of a card's fields.
type FieldValue struct {
	Field string `json:"field" jsonschema:"the field's name, spelled as the stencil declares it"`
	Text  string `json:"text" jsonschema:"the HTML under that heading"`
}

// Fault is one thing wrong with a deck, on the card it is against.
type Fault struct {
	Card int    `json:"card" jsonschema:"where the card stands in the deck, counted from its first card; -1 for a fault against no card"`
	Why  string `json:"why" jsonschema:"what is wrong, and what the application did about it"`
}

// CurrentCard is the card in front of the person, as an agent is told about it. The deck is the file
type CurrentCard struct {
	Deck string `json:"deck" jsonschema:"the deck the card stands in, by the path the vault files it under; empty when no card is in front of them"`
	Card string `json:"card,omitempty" jsonschema:"the card's mark, as card_read gives it"`
	Face string `json:"face,omitempty" jsonschema:"the face it is being shown through, spelled as the stencil writes it"`
}

// Face is one way a stencil shows a card.
type Face struct {
	Name  string `json:"name" jsonschema:"what the face is called"`
	Front string `json:"front" jsonschema:"what is shown before the answer, as HTML with {{Field}} where a value goes"`
	Back  string `json:"back" jsonschema:"what is shown after it"`
}

// WriteOutcome is what a deck became: where it is, and what to present at the next write of it.
type WriteOutcome struct {
	Path        string `json:"path"`
	Fingerprint string `json:"fingerprint" jsonschema:"hand this to the next write of this deck without reading it back"`
	Cards       int    `json:"cards" jsonschema:"how many cards the deck now holds"`
	Mark        string `json:"mark,omitempty" jsonschema:"the mark the card just written is addressed by, for as long as it exists"`
}

func newWriteOutcome(path string, edited deckedit.EditResult) WriteOutcome {
	return WriteOutcome{Path: path, Fingerprint: fingerprintOf(edited.Fingerprint), Cards: edited.Cards}
}

func newCard(card format.Card) Card {
	out := Card{Mark: string(card.Mark), Section: card.Section, Stencil: card.StencilLink}
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

func newFaceTemplates(fs []Face) []format.FaceTemplate {
	out := make([]format.FaceTemplate, 0, len(fs))
	for _, f := range fs {
		out = append(out, format.FaceTemplate{Name: f.Name, Front: f.Front, Back: f.Back})
	}
	return out
}

func newFaces(templates []format.FaceTemplate) []Face {
	out := make([]Face, 0, len(templates))
	for _, t := range templates {
		out = append(out, Face{Name: t.Name, Front: t.Front, Back: t.Back})
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

// fingerprintOf is what a deck or a stencil was when it was read, as an agent hands it back.
func fingerprintOf(ref domain.Fingerprint) string {
	return strconv.FormatInt(ref.Size, 10) + "-" + strconv.FormatInt(stamp(ref.ModTime), 10)
}

// parseFingerprint is the fingerprint a caller presents. A tool that changes what somebody may have
func parseFingerprint(s string) (domain.Fingerprint, error) {
	if s == "" {
		return domain.Fingerprint{}, errors.New("present the fingerprint the read gave you: card_read for a deck, card_stencil_read for a stencil")
	}
	size, mtime, found := strings.Cut(s, "-")
	if !found {
		return domain.Fingerprint{}, fmt.Errorf("%q is not a fingerprint a read gave out", s)
	}
	ref := domain.Fingerprint{}
	var err error
	if ref.Size, err = strconv.ParseInt(size, 10, 64); err != nil {
		return domain.Fingerprint{}, fmt.Errorf("%q is not a fingerprint a read gave out", s)
	}
	nanos, err := strconv.ParseInt(mtime, 10, 64)
	if err != nil {
		return domain.Fingerprint{}, fmt.Errorf("%q is not a fingerprint a read gave out", s)
	}
	ref.ModTime = instant(nanos)
	return ref, nil
}

func convertOutcomeToErrorCode(o edit.ReadOutcome) (v1.ErrorCode, bool) {
	switch o {
	case edit.Missing:
		return v1.ErrorCode_ERROR_CODE_MISSING, true
	case edit.NotANote:
		return v1.ErrorCode_ERROR_CODE_NOT_A_NOTE, true
	case edit.NotText:
		return v1.ErrorCode_ERROR_CODE_NOT_TEXT, true
	case edit.TooLarge:
		return v1.ErrorCode_ERROR_CODE_TOO_LARGE, true
	case edit.Unreadable:
		return v1.ErrorCode_ERROR_CODE_UNREADABLE, true
	case edit.Ok:
		return v1.ErrorCode_ERROR_CODE_UNSPECIFIED, false
	}
	return v1.ErrorCode_ERROR_CODE_UNSPECIFIED, false
}

func describeCode(reason v1.ErrorCode) string {
	switch reason {
	case v1.ErrorCode_ERROR_CODE_MISSING:
		return "the vault holds no note at this path"
	case v1.ErrorCode_ERROR_CODE_NOT_A_NOTE:
		return "this is not a note the vault holds"
	case v1.ErrorCode_ERROR_CODE_NOT_TEXT:
		return "this file is not text: some of it is not valid UTF-8, so open it as a file"
	case v1.ErrorCode_ERROR_CODE_TOO_LARGE:
		return fmt.Sprintf("it is larger than the %d bytes this reads; open the file instead", edit.MaxBytes)
	case v1.ErrorCode_ERROR_CODE_BODY_UNWRITABLE:
		return "this text opens with a frontmatter delimiter, so it cannot be written into a note"
	case v1.ErrorCode_ERROR_CODE_UNREADABLE:
		return "the frontmatter of this note cannot be read, so it can be neither read nor written from here"
	case v1.ErrorCode_ERROR_CODE_OCCUPIED:
		return "a file is already where this note would go"
	case v1.ErrorCode_ERROR_CODE_UNNAMEABLE:
		return "a note cannot be called this"
	case v1.ErrorCode_ERROR_CODE_NOT_A_STENCIL:
		return "this note is not a stencil"
	case v1.ErrorCode_ERROR_CODE_NOT_A_DECK:
		return "this note is not a deck"
	case v1.ErrorCode_ERROR_CODE_DECK_TOO_LARGE:
		return "this deck is larger than a deck is read at"
	case v1.ErrorCode_ERROR_CODE_NOT_A_PRESET:
		return "this note is not a preset"
	case v1.ErrorCode_ERROR_CODE_STALE:
		return "this file is no longer the one that was read, and nothing was written: read it again and write from what is there now"
	default:
		return reason.String()
	}
}
