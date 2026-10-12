package mcp

import (
	"context"
	"errors"
	"fmt"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// addViewTools adds the tools that move the person's window to a note or a source passage.
// They are added only when the core has a view.
func addViewTools(server *sdk.Server, core Core) {
	if core.View == nil {
		return
	}
	addNoteFocus(server, core)
	addSourceFocus(server, core)
}

type noteFocusInput struct {
	Path string `json:"path" jsonschema:"the note to put in focus, by its path relative to the vault folder"`
}

type noteFocusOutput struct {
	Focused Note `json:"focused"`
}

func addNoteFocus(server *sdk.Server, core Core) {
	addTool(server, &sdk.Tool{
		Name:  "note_focus",
		Title: "Focus a note",
		Description: "Make a note the one the person is looking at, so that the " +
			"neighbourhood they see is drawn around it. Use it only where the person " +
			"asked to be taken to a note. Talking about one is not asking, and neither " +
			"is changing one: the person may be working somewhere else, and this moves " +
			"them. It reads nothing back: `note_read` does that.",
	}, newHandler(func(ctx context.Context, in noteFocusInput) (noteFocusOutput, error) {
		return focusNote(ctx, core, in)
	}))
}

func focusNote(ctx context.Context, core Core, in noteFocusInput) (noteFocusOutput, error) {
	if in.Path == "" {
		return noteFocusOutput{}, errors.New("name the note to put in focus")
	}

	found, err := core.Notes.Queries.Notes(ctx, core.getCurrentVault().Vault.ID, []string{in.Path})
	if err != nil {
		return noteFocusOutput{}, err
	}
	ref, known := found[in.Path]
	if !known {
		return noteFocusOutput{}, fmt.Errorf("no note at %s", in.Path)
	}
	if err := core.View.Focus(ctx, domain.Place{Path: in.Path}); err != nil {
		return noteFocusOutput{}, err
	}
	return noteFocusOutput{Focused: noteOf(ref)}, nil
}

type passageSpan struct {
	Start  int `json:"start"`
	Length int `json:"length"`
}

type sourceFocusInput struct {
	Path   string        `json:"path" jsonschema:"the document to open, by its path relative to the vault folder"`
	Start  int           `json:"start" jsonschema:"where the passage begins in the document's text, in bytes, as a search gives it"`
	Length int           `json:"length" jsonschema:"how long the passage is, in bytes, as a search gives it; zero opens the document at its beginning"`
	Also   []passageSpan `json:"also,omitempty" jsonschema:"the other passages of the same document to light, as a search gives them"`
}

type sourceFocusOutput struct {
	IsShown bool   `json:"shown"`
	Looking string `json:"looking" jsonschema:"what the person is now looking at, in words to say back to them"`
}

func addSourceFocus(server *sdk.Server, core Core) {
	addTool(server, &sdk.Tool{
		Name:  "source_focus",
		Title: "Focus a passage",
		Description: "Open one of the vault's documents in front of the person at one " +
			"passage: the page it stands on is drawn, and the words of it are lit. " +
			"`note_search` gives the range of every passage it answers with, and this " +
			"takes that range as it stands. Where a question is answered in several " +
			"places of one document, name the rest under `also`: the person is taken " +
			"to the first and the others are lit where they fall. Use it only where the " +
			"person asked to be shown something in a book. Talking about a passage is " +
			"not asking: the person may be working somewhere else, and this moves them.",
	}, newHandler(func(ctx context.Context, in sourceFocusInput) (sourceFocusOutput, error) {
		return focusSource(ctx, core, in)
	}))
}

func focusSource(ctx context.Context, core Core, in sourceFocusInput) (sourceFocusOutput, error) {
	if err := checkSourceFocus(in); err != nil {
		return sourceFocusOutput{}, err
	}
	ref, err := getFingerprint(ctx, core, in.Path)
	if err != nil {
		return sourceFocusOutput{}, err
	}
	at, err := newPlace(in)
	if err != nil {
		return sourceFocusOutput{}, err
	}
	if err := core.View.Focus(ctx, at); err != nil {
		return sourceFocusOutput{}, err
	}
	return sourceFocusOutput{IsShown: true, Looking: describeFocus(ref.Kind, in.Length)}, nil
}

// checkSourceFocus returns an error for an input with no document, a passage that
// starts before the text, or more passages than a document may highlight.
func checkSourceFocus(in sourceFocusInput) error {
	if in.Path == "" {
		return errors.New("name the document to show")
	}
	if in.Start < 0 || in.Length < 0 {
		return errors.New(
			"a passage begins at or after the start of the text, and its length is zero or more")
	}
	if 1+len(in.Also) > domain.MostHighlights {
		return fmt.Errorf("light at most %d places of one document", domain.MostHighlights)
	}
	return nil
}

// newPlace returns the place for the document and passages named in the input.
func newPlace(in sourceFocusInput) (domain.Place, error) {
	at := domain.Place{Path: in.Path}
	if in.Length > 0 {
		at.Spans = append(at.Spans, domain.ByteSpan{From: in.Start, To: in.Start + in.Length})
	}
	for _, one := range in.Also {
		if one.Start < 0 || one.Length <= 0 {
			return domain.Place{}, errors.New(
				"a passage begins at or after the start of the text, and is longer than nothing")
		}
		at.Spans = append(at.Spans, domain.ByteSpan{From: one.Start, To: one.Start + one.Length})
	}
	return at, nil
}

// getFingerprint returns the fingerprint of the file at path, and an error when the vault
// holds no file there.
func getFingerprint(ctx context.Context, core Core, path string) (domain.Fingerprint, error) {
	if core.Readers == nil {
		return domain.Fingerprint{}, errors.New("this vault's files are not open")
	}
	reader, err := core.Readers.Open(core.getCurrentVault().Vault)
	if err != nil {
		return domain.Fingerprint{}, err
	}
	ref, err := reader.Stat(ctx, path)
	if port.IsNoNote(err) {
		return domain.Fingerprint{}, fmt.Errorf("this vault holds nothing at %s", path)
	}
	if err != nil {
		return domain.Fingerprint{}, err
	}
	return ref, nil
}

// describeFocus returns a sentence describing what the person now sees.
func describeFocus(kind domain.SourceKind, length int) string {
	switch {
	case kind == domain.KindNote:
		return "the note is in front of them, and a note is shown whole"
	case length == 0:
		return "the document is open at its beginning"
	}
	return "the document is open at that passage, with the words of it lit"
}
