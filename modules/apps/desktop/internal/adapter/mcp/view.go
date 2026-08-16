package mcp

import (
	"context"
	"errors"
	"fmt"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// addViewTools gives an agent the one thing a person watching can do that
// reading and writing cannot: put a note in front of them.
//
// They are added only where there is somebody to show a note to. A binary
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
		if err := core.View.Focus(ctx, in.Path); err != nil {
			return nil, out{}, err
		}
		return nil, out{Focused: noteOf(ref)}, nil
	})
}
