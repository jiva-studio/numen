package mcp

import (
	"context"
	"errors"
	"fmt"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// NewURL is one address a caller wants a file made for: where it goes, and
// whether the video at that address is wanted on this disk.
type NewURL struct {
	URL        string `json:"url" jsonschema:"the address to import, as a browser would go to it"`
	Folder     string `json:"folder,omitempty" jsonschema:"where to file it, relative to the vault folder; the root by default"`
	ShouldCopy bool   `json:"copy,omitempty" jsonschema:"fetch the video itself onto this disk as well; a copy is large, so ask only when it is wanted"`
}

// ImportOutcome is the file that now exists and what was fetched for it.
type ImportOutcome struct {
	Path string `json:"path" jsonschema:"where the file stands in the vault; what it is called is what is at the address, once that is known"`
	// Producer is what brought the text back, and Bytes how much of it there
	// is. Both are empty where the address published none of what was asked for.
	Producer  string `json:"producer,omitempty" jsonschema:"what fetched the text: captions for a video's words, article for a page's prose"`
	Bytes     int    `json:"bytes,omitempty" jsonschema:"how much text came back, in bytes"`
	IsNothing bool   `json:"nothing,omitempty" jsonschema:"the address publishes none of what was asked for, which is an answer and not a failure"`
	// CopiedBytes is how large the copy is, and is nothing where none was asked
	// for or the video was over the size the settings allow.
	CopiedBytes int64  `json:"copied_bytes,omitempty" jsonschema:"how large the copy on this disk is"`
	CopyError   string `json:"copy_error,omitempty" jsonschema:"why no copy was made, empty when one was or none was asked for"`
	Error       string `json:"error,omitempty" jsonschema:"why the address was not fetched; the file stands either way"`
}

// addImportTool serves the one tool that makes a file out of an address.
func addImportTool(server *sdk.Server, core Core) {
	if core.Sources.CreateAndImportURL == nil {
		return
	}
	addTool(server, &sdk.Tool{
		Name:  "url_import",
		Title: "Import a url",
		Description: "Make a file holding a web address and fetch what is there. " +
			"A video's published words come back as words with the times they were " +
			"said at, and any other page comes back as the prose it is written " +
			"around; either is searched from then on and read by `artifact_read`. " +
			"The file is named after what is at the address, so no title is asked " +
			"for.\n\nIt holds the address and nothing else: what you write about " +
			"what is there is a note of your own, pointing at this file. `copy` " +
			"also fetches the video itself onto this disk, which is a large file " +
			"and is worth asking for only when somebody wants to watch it without " +
			"the site. An address nothing here reaches, and one that refuses an " +
			"unattended request, say so and leave the file standing.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in NewURL) (*sdk.CallToolResult, ImportOutcome, error) {
		pos, err := domain.ParseURL(in.URL)
		if err != nil {
			return nil, ImportOutcome{}, err
		}
		imported, err := core.Sources.CreateAndImportURL.Execute(
			ctx, core.getCurrentVault().Vault, source.NewURL{Address: pos, Path: in.Folder}, in.ShouldCopy)
		out := ImportOutcome{Path: imported.Path}
		var copyFailed *source.CopyError
		if err != nil && !errors.As(err, &copyFailed) {
			out.Error = sayError(err)
			return nil, out, nil
		}
		out.Producer, out.Bytes, out.IsNothing = imported.Producer, imported.Bytes, imported.IsNothing
		if in.ShouldCopy {
			out.CopiedBytes, out.CopyError = describeCopy(imported.Copy, err)
		}
		return nil, out, nil
	})
}

// describeCopy says how large the copy of a video is, and why where there is none.
func describeCopy(got source.CopyResult, err error) (int64, string) {
	switch {
	case errors.Is(err, source.ErrNotAURL):
		return 0, "there is nothing to copy at that address"
	case errors.Is(err, source.ErrBeingDownloaded):
		return 0, "another run is downloading this address"
	case err != nil:
		return 0, sayError(err)
	case got.IsTooLarge():
		return 0, fmt.Sprintf(
			"it is %d MB, over the %d MB importing.copy_max_size_mb allows",
			got.Bytes>>20, got.Limit>>20)
	}
	return got.Bytes, ""
}
