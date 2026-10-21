package mcp

import (
	"context"
	"fmt"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/modules/libs/core/usecase/recognition"
)

func addSourceReadTools(server *sdk.Server, core Core) {
	addSourceList(server, core)
	addSourceRead(server, core)
}

type sourceListOutput struct {
	Documents   []Source `json:"documents"`
	Recognition string   `json:"recognition,omitempty" jsonschema:"what stands in the way of having a scan read now, when anything does"`
}

func addSourceList(server *sdk.Server, core Core) {
	addTool(server, &sdk.Tool{
		Name:  "source_list",
		Title: "List documents",
		Description: "List the books, papers and scans filed in the vault beside its notes, " +
			"and say of each whether a model has read it. A scanned document carries no " +
			"text of its own until it is read, and nothing in the words of a document " +
			"that does carry text says whether that text is any good. Use this before " +
			"asking for one to be read.",
	}, newHandler(func(ctx context.Context, _ struct{}) (sourceListOutput, error) {
		return listSources(ctx, core)
	}))
}

func listSources(ctx context.Context, core Core) (sourceListOutput, error) {
	if core.Sources.GetDocumentList == nil {
		return sourceListOutput{}, fmt.Errorf("this vault's sources are not open")
	}
	items, err := core.Sources.GetDocumentList.Execute(ctx, core.getCurrentVault().Vault)
	if err != nil {
		return sourceListOutput{}, err
	}
	documents := make([]Source, 0, len(items))
	for _, one := range items {
		documents = append(documents, Source{Path: one.Path, ShouldRead: one.IsRead})
	}
	return sourceListOutput{Documents: documents, Recognition: getRecognitionBlocker(core.Sources.GetRecognitionStatus)}, nil
}

// getRecognitionBlocker returns the reason a scan cannot be recognised now, empty when none.
func getRecognitionBlocker(getStatus *recognition.GetStatus) string {
	if getStatus == nil {
		return ""
	}
	status := getStatus.Execute()
	switch {
	case status.IsRunning:
		return "one document is being read now"
	case !status.IsReady:
		return "what is needed to read scans is not here yet, and is fetched when one is asked for"
	}
	return ""
}

type sourceReadInput struct {
	Path   string `json:"path" jsonschema:"the document, as source_list gives it"`
	Start  int    `json:"start" jsonschema:"where the run begins in the document's text, in bytes"`
	Length int    `json:"length" jsonschema:"how much to read, in bytes"`
}

type sourceReadOutput struct {
	Text     string `json:"text"`
	Location string `json:"location,omitempty" jsonschema:"where the run begins in the document's own numbering"`
	Start    int    `json:"start" jsonschema:"where the run begins, which is what was asked for held within the text"`
	Length   int    `json:"length" jsonschema:"how long the run is"`
	Size     int    `json:"size" jsonschema:"how long the document's whole text is, in bytes, so what stands on either side can be asked for"`
}

func addSourceRead(server *sdk.Server, core Core) {
	addTool(server, &sdk.Tool{
		Name:  "source_read",
		Title: "Read a range of a document",
		Description: "Read a span of one document's own text, in the offsets a search's " +
			"passage carries. A passage is a window cut to a size and it ends where it " +
			"was cut, so what answers the question often stands just past it: ask for " +
			"the run beginning at the passage's start plus its length to read on, or " +
			"for one beginning before it to read back. The text is what the reader that " +
			"made it wrote, corrections and all, which is the text a search matched. " +
			"Notes are read with note_read.",
	}, newHandler(func(ctx context.Context, in sourceReadInput) (sourceReadOutput, error) {
		return readSource(ctx, core, in)
	}))
}

func readSource(ctx context.Context, core Core, in sourceReadInput) (sourceReadOutput, error) {
	if core.Sources.GetDocumentText == nil {
		return sourceReadOutput{}, fmt.Errorf("this vault's sources are not open")
	}
	res, err := core.Sources.GetDocumentText.Execute(ctx, core.getCurrentVault().Vault, in.Path, in.Start, in.Length)
	if err != nil {
		return sourceReadOutput{}, err
	}
	return sourceReadOutput{
		Text:     res.Text,
		Location: res.Location,
		Start:    res.Start,
		Length:   res.Length,
		Size:     res.Whole,
	}, nil
}
