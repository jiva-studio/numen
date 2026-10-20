package mcp

import (
	"context"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// addViewTools adds the tools that move the person's window to a note or a source passage.
// Each is added only when the core can put that in front of the person.
func addViewTools(server *sdk.Server, core Core) {
	if core.Notes.FocusNote != nil {
		addNoteFocus(server, core)
	}
	if core.Sources.FocusPassage != nil {
		addSourceFocus(server, core)
	}
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
	ref, err := core.Notes.FocusNote.Execute(ctx, core.getCurrentVault().Vault, in.Path)
	if err != nil {
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
	Summary string `json:"summary" jsonschema:"what the person is now looking at, in words to say back to them"`
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
	passage := source.Passage{Path: in.Path, Start: in.Start, Length: in.Length}
	for _, one := range in.Also {
		passage.Also = append(passage.Also, domain.ByteSpan{From: one.Start, To: one.Start + one.Length})
	}
	summary, err := core.Sources.FocusPassage.Execute(ctx, core.getCurrentVault().Vault, passage)
	if err != nil {
		return sourceFocusOutput{}, err
	}
	return sourceFocusOutput{IsShown: true, Summary: summary}, nil
}
