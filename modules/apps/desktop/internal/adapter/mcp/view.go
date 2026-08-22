package mcp

import (
	"context"
	"errors"
	"fmt"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
)

// mostPlaces is how many places beyond the one the person is taken to may be
// lit at once. What is lit at once is what a person can take in.
const mostPlaces = 7

// addViewTools gives an agent the one thing a person watching can do that
// reading and writing cannot: put a place in front of them.
//
// They are added only where there is somebody to show a place to. A binary
// without a window serves the vault and nothing of this.
func addViewTools(server *sdk.Server, core Core) {
	if core.View == nil {
		return
	}

	sdk.AddTool(server, &sdk.Tool{
		Name:  "note_focus",
		Title: "Put a note in front of the person",
		Description: "Make a note the one the person is looking at, so that the " +
			"neighbourhood they see is drawn around it. Use it while talking about " +
			"a note, or after changing one. It reads nothing back: `note_read` does that.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		Path string `json:"path" jsonschema:"the note to put in focus, by its path relative to the vault folder"`
	}) (*sdk.CallToolResult, struct {
		Focused Note `json:"focused"`
	}, error) {
		type out = struct {
			Focused Note `json:"focused"`
		}
		if in.Path == "" {
			return nil, out{}, errors.New("name the note to put in focus")
		}

		found, err := core.Notes.Notes(ctx, core.Vault.ID, []string{in.Path})
		if err != nil {
			return nil, out{}, err
		}
		ref, known := found[in.Path]
		if !known {
			return nil, out{}, fmt.Errorf("no note at %s", in.Path)
		}
		if err := core.View.Focus(ctx, domain.Place{Path: in.Path}); err != nil {
			return nil, out{}, err
		}
		return nil, out{Focused: noteOf(ref)}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:  "source_show",
		Title: "Show the person a passage of a document",
		Description: "Open one of the vault's documents in front of the person at one " +
			"passage: the page it stands on is drawn, and the words of it are lit. " +
			"`note_search` gives the range of every passage it answers with, and this " +
			"takes that range as it stands. Where a question is answered in several " +
			"places of one document, name the rest under `also`: the person is taken " +
			"to the first and the others are lit where they fall. Use it when somebody " +
			"asks to be shown something in a book, or to put what you are talking " +
			"about in front of them.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		Path   string `json:"path" jsonschema:"the document to open, by its path relative to the vault folder"`
		Start  int    `json:"start" jsonschema:"where the passage begins in the document's text, in bytes, as a search gives it"`
		Length int    `json:"length" jsonschema:"how long the passage is, in bytes, as a search gives it; zero opens the document at its beginning"`
		Also   []struct {
			Start  int `json:"start"`
			Length int `json:"length"`
		} `json:"also,omitempty" jsonschema:"the other passages of the same document to light, as a search gives them"`
	}) (*sdk.CallToolResult, struct {
		Shown bool   `json:"shown"`
		Says  string `json:"says"`
	}, error) {
		type out = struct {
			Shown bool   `json:"shown"`
			Says  string `json:"says"`
		}
		if in.Path == "" {
			return nil, out{}, errors.New("name the document to show")
		}
		if in.Start < 0 || in.Length < 0 {
			return nil, out{}, errors.New(
				"a passage begins at or after the start of the text, and its length is zero or more")
		}

		if len(in.Also) > mostPlaces {
			return nil, out{}, fmt.Errorf("light at most %d places of one document", mostPlaces+1)
		}

		ref, err := holding(ctx, core, in.Path)
		if err != nil {
			return nil, out{}, err
		}
		at := domain.Place{Path: in.Path, Start: in.Start, Length: in.Length}
		for _, one := range in.Also {
			if one.Start < 0 || one.Length <= 0 {
				return nil, out{}, errors.New(
					"a passage begins at or after the start of the text, and is longer than nothing")
			}
			at.Also = append(at.Also, domain.Stretch{Start: one.Start, Length: one.Length})
		}
		if err := core.View.Focus(ctx, at); err != nil {
			return nil, out{}, err
		}
		return nil, out{Shown: true, Says: showing(ref.Kind, in.Length)}, nil
	})
}

// holding is what the vault holds at a path, and says so when it holds nothing
// there. A file the vault leaves alone is a file it does not hold.
func holding(ctx context.Context, core Core, path string) (domain.FileRef, error) {
	if core.Readers == nil {
		return domain.FileRef{}, errors.New("this vault's files are not open")
	}
	reader, err := core.Readers.Open(core.Vault)
	if err != nil {
		return domain.FileRef{}, err
	}
	ref, err := reader.Stat(ctx, path)
	if port.NoNote(err) {
		return domain.FileRef{}, fmt.Errorf("this vault holds nothing at %s", path)
	}
	if err != nil {
		return domain.FileRef{}, err
	}
	return ref, nil
}

// showing is what the person now has in front of them, for the agent to say
// back to them.
func showing(kind domain.SourceKind, length int) string {
	switch {
	case kind == domain.KindNote:
		return "the note is in front of them, and a note is shown whole"
	case length == 0:
		return "the document is open at its beginning"
	}
	return "the document is open at that passage, with the words of it lit"
}
