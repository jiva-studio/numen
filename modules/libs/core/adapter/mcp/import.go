package mcp

import (
	"context"
	"errors"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
)

// NewLinkNote is one note a caller wants made out of an address: where it goes,
// and whether the video at that address is wanted on this disk.
type NewLinkNote struct {
	URL    string `json:"url" jsonschema:"the address to import, as a browser would go to it"`
	Folder string `json:"folder,omitempty" jsonschema:"where to file the note, relative to the vault folder; the root by default"`
	Body   string `json:"body,omitempty" jsonschema:"the markdown to start the note with, above what is fetched"`
	Copy   bool   `json:"copy,omitempty" jsonschema:"fetch the video itself onto this disk as well; a copy is large, so ask only when it is wanted"`
}

// ImportOutcome is the note that now exists and what was fetched into it.
type ImportOutcome struct {
	Path  string `json:"path" jsonschema:"where the note stands in the vault"`
	Title string `json:"title" jsonschema:"what the note is called, which is what is at the address once that is known"`
	// Producer is what brought the text back, and Words how much of it there
	// is. Both are empty where the address published none of what was asked for.
	Producer string `json:"producer,omitempty" jsonschema:"what fetched the text: captions for a video's words, article for a page's prose"`
	Words    int    `json:"words,omitempty" jsonschema:"how much text came back, in bytes"`
	Nothing  bool   `json:"nothing,omitempty" jsonschema:"the address publishes none of what was asked for, which is an answer and not a failure"`
	// CopiedBytes is how large the copy is, and is nothing where none was asked
	// for or the video was over the size the settings allow.
	CopiedBytes int64  `json:"copied_bytes,omitempty" jsonschema:"how large the copy on this disk is"`
	CopyRefused string `json:"copy_refused,omitempty" jsonschema:"why no copy was made, empty when one was or none was asked for"`
	Refused     string `json:"refused,omitempty" jsonschema:"why the address was not fetched; the note stands either way"`
}

// addImportTool serves the one tool that makes a note out of an address.
func addImportTool(server *sdk.Server, core Core) {
	if core.Notes.Import == nil {
		return
	}
	sdk.AddTool(server, &sdk.Tool{
		Name:  "note_import",
		Title: "Import an address",
		Description: "Make a note pointing at a web address and fetch what is there. " +
			"A video's published words come back as words with the times they were " +
			"said at, and any other page comes back as the prose it is written " +
			"around; either is searched with the note from then on and read by " +
			"`note_read`. The note is named after what is at the address, so no " +
			"title is asked for. `copy` also fetches the video itself onto this " +
			"disk, which is a large file and is worth asking for only when somebody " +
			"wants to watch it without the site. An address nothing here reaches, " +
			"and one that refuses an unattended request, say so and leave the note " +
			"pointing where it points.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in NewLinkNote) (*sdk.CallToolResult, ImportOutcome, error) {
		at, err := domain.ParseWebAddress(in.URL)
		if err != nil {
			return nil, ImportOutcome{}, err
		}
		v := core.shown().Vault
		// The note is named by the address until what is there says what it is
		// called, which is what the import does next.
		made, err := core.Notes.Create.Execute(ctx, v, note.NewNote{
			Title: at.URL, Body: in.Body, Folder: in.Folder, Address: at,
		})
		if err != nil {
			return nil, ImportOutcome{Path: made.Path, Title: at.URL, Refused: refusing(err)}, nil
		}

		out := ImportOutcome{Path: made.Path, Title: made.Title}
		fetched, err := core.Notes.Import.Execute(ctx, v, made.Path)
		if err != nil {
			out.Refused = refusing(err)
			return nil, out, nil
		}
		out.Path, out.Producer = fetched.Path, fetched.Producer
		out.Words, out.Nothing = fetched.Words, fetched.Nothing
		if fetched.Title != "" {
			out.Title = fetched.Title
		}
		if in.Copy {
			out.CopiedBytes, out.CopyRefused = copying(ctx, core, v, out.Path)
		}
		return nil, out, nil
	})
}

// copying fetches the video at a note's address, and says why where it did not.
func copying(
	ctx context.Context, core Core, v domain.Vault, path string,
) (int64, string) {
	got, err := core.Notes.Import.Copy(ctx, v, path)
	switch {
	case errors.Is(err, source.ErrNotALink):
		return 0, "there is no video at that address"
	case err != nil:
		return 0, refusing(err)
	case got.TooLarge:
		return 0, "the video is larger than importing.copy_under_mb"
	case got.Busy:
		return 0, "another run is fetching this video now"
	}
	return got.Bytes, ""
}
