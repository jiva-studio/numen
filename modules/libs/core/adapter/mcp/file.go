package mcp

import (
	"context"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/modules/libs/core/usecase/file"
)

// addFileReadingTools adds the tools that reach a vault's files by path,
// whatever kind of file the vault holds each of them as.
func addFileReadingTools(server *sdk.Server, core Core) {
	sdk.AddTool(server, &sdk.Tool{
		Name:  "file_read",
		Title: "Read a run of a file",
		Description: "Read a stretch of any file the vault holds, by its path from the " +
			"vault folder. Reach for this when a path is named and the file behind it is " +
			"neither a note nor a document the vault has read — a transcript somebody " +
			"typed, an export, whatever they put in the folder — and when a file is long " +
			"enough that reading it whole would take the answer: ask for the run " +
			"beginning at the previous run's start plus its length to read on. The answer " +
			"says how long the whole file is. A note's prose, without its frontmatter, is " +
			"note_read; the text of a book or a recording is source_read.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		Path   string `json:"path" jsonschema:"the file's path relative to the vault folder, with forward slashes"`
		Start  int    `json:"start,omitempty" jsonschema:"where the run begins in the file, in bytes; the beginning of the file by default"`
		Length int    `json:"length,omitempty" jsonschema:"how much to read, in bytes; as much as one call carries by default"`
	}) (*sdk.CallToolResult, struct {
		Text    string `json:"text,omitempty"`
		Start   int    `json:"start" jsonschema:"where the run begins, which is what was asked for held within the file"`
		Length  int    `json:"length" jsonschema:"how long the run is"`
		Size    int    `json:"size" jsonschema:"how long the whole file is, in bytes"`
		Refused string `json:"refused,omitempty" jsonschema:"why nothing came back, empty when the run did"`
	}, error) {
		type out = struct {
			Text    string `json:"text,omitempty"`
			Start   int    `json:"start" jsonschema:"where the run begins, which is what was asked for held within the file"`
			Length  int    `json:"length" jsonschema:"how long the run is"`
			Size    int    `json:"size" jsonschema:"how long the whole file is, in bytes"`
			Refused string `json:"refused,omitempty" jsonschema:"why nothing came back, empty when the run did"`
		}
		contents, err := file.Read{Readers: core.Readers}.Execute(
			ctx, core.shown().Vault, in.Path, in.Start, in.Length)
		if err != nil {
			return nil, out{}, err
		}
		res := out{Start: contents.Start, Length: contents.Length, Size: contents.Whole}
		if contents.Outcome == file.Ok {
			res.Text = contents.Text
			return nil, res, nil
		}
		res.Refused = unread(contents.Outcome)
		return nil, res, nil
	})
}

// unread is a file read's outcome in words an agent can act on. A folder and a
// file the vault passes over are outcomes only a file has, and the Refusal the
// windows are answered with names neither.
func unread(outcome file.ReadOutcome) string {
	switch outcome {
	case file.Missing:
		return "the vault holds no file at this path"
	case file.LeftAlone:
		return "the vault passes this file over, so nothing here reads it"
	case file.AFolder:
		return "this path holds a folder"
	case file.NotText:
		return "this run of the file is not text: some of it is not valid UTF-8"
	default:
		return string(outcome)
	}
}
