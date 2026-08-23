package mcp

import (
	"context"
	"fmt"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/source"
)

// A Document is one source of a vault that is not a note, as an agent is told
// about it.
type Document struct {
	Path string `json:"path"`
	// Read says whether this document stands on what a model read in it. A
	// reading still running stands on the pages it has reached, so this is true
	// from the first of them.
	//
	// A scan that carries its own text says nothing about whether that text is
	// any good, so this is the only thing that can be said for certain.
	Read bool `json:"read"`
}

// addSourceTools adds the tools for the sources a vault holds beside its notes.
//
// There are two, and the first is why the second is usable: an agent asked to
// read a document has to be able to find out which have been read already, or
// it will read one twice and never read another.
func addSourceTools(server *sdk.Server, core Core) {
	sdk.AddTool(server, &sdk.Tool{
		Name:  "source_list",
		Title: "List the documents a vault holds",
		Description: "List the books, papers and scans filed in the vault beside its notes, " +
			"and say of each whether a model has read it. A scanned document carries no " +
			"text of its own until it is read, and nothing in the words of a document " +
			"that does carry text says whether that text is any good. Use this before " +
			"asking for one to be read.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, _ struct{}) (*sdk.CallToolResult, struct {
		Documents []Document `json:"documents"`
		Says      string     `json:"says,omitempty"`
	}, error) {
		type out = struct {
			Documents []Document `json:"documents"`
			Says      string     `json:"says,omitempty"`
		}
		if core.Sources == nil {
			return nil, out{}, fmt.Errorf("this vault's sources are not open")
		}
		known, err := core.Sources.Fingerprints(ctx, core.Vault.ID, domain.KindBook)
		if err != nil {
			return nil, out{}, err
		}
		read, err := core.Sources.Recognised(ctx, core.Vault.ID, domain.KindBook)
		if err != nil {
			return nil, out{}, err
		}
		stands := make(map[string]bool, len(read))
		for _, one := range read {
			stands[one.Path] = one.From != ""
		}
		documents := make([]Document, 0, len(known))
		for path := range known {
			documents = append(documents, Document{Path: path, Read: stands[path]})
		}
		says := ""
		if core.Recognise != nil {
			switch {
			case core.Recognise.Running():
				says = "one document is being read now"
			case !core.Recognise.Ready():
				says = "what is needed to read scans is not here yet, and is fetched when one is asked for"
			}
		}
		return nil, out{Documents: documents, Says: says}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:  "source_read",
		Title: "Read a run of a document's text",
		Description: "Read a stretch of one document's own text, in the offsets a search's " +
			"passage carries. A passage is a window cut to a size and it ends where it " +
			"was cut, so what answers the question often stands just past it: ask for " +
			"the run beginning at the passage's start plus its length to read on, or " +
			"for one beginning before it to read back. The text is what the reader that " +
			"made it wrote, corrections and all, which is the text a search matched. " +
			"Notes are read with note_read.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		Path   string `json:"path" jsonschema:"the document, as source_list gives it"`
		Start  int    `json:"start" jsonschema:"where the run begins in the document's text, in bytes"`
		Length int    `json:"length" jsonschema:"how much to read, in bytes"`
	}) (*sdk.CallToolResult, struct {
		Text     string `json:"text"`
		Location string `json:"location,omitempty" jsonschema:"where the run begins in the document's own numbering"`
		Start    int    `json:"start" jsonschema:"where the run begins, which is what was asked for held within the text"`
		Length   int    `json:"length" jsonschema:"how long the run is"`
		Whole    int    `json:"whole" jsonschema:"how long the document's text is, so what stands on either side can be asked for"`
	}, error) {
		type out = struct {
			Text     string `json:"text"`
			Location string `json:"location,omitempty" jsonschema:"where the run begins in the document's own numbering"`
			Start    int    `json:"start" jsonschema:"where the run begins, which is what was asked for held within the text"`
			Length   int    `json:"length" jsonschema:"how long the run is"`
			Whole    int    `json:"whole" jsonschema:"how long the document's text is, so what stands on either side can be asked for"`
		}
		if core.Sources == nil {
			return nil, out{}, fmt.Errorf("this vault's sources are not open")
		}
		res, err := source.Read{
			Readers: core.Readers,
			Sources: core.Sources,
			Derived: core.Derived,
		}.Execute(ctx, core.Vault, in.Path, in.Start, in.Length)
		if err != nil {
			return nil, out{}, err
		}
		return nil, out{
			Text:     res.Text,
			Location: res.Location,
			Start:    res.Start,
			Length:   res.Length,
			Whole:    res.Whole,
		}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:  "source_recognise",
		Title: "Read a scanned document",
		Description: "Have a model read one scanned document and write down what it says. " +
			"The pages it has read are searchable as it goes, so a search finds the " +
			"beginning of a book long before the end of it is read; the part not yet " +
			"read answers nothing until it is. This is slow — an hour for a book — and " +
			"it is never done on its own, because whether a document's own text is any " +
			"good cannot be told from the text. Ask for it when a document is a scan, " +
			"or when what a search returns from one is nonsense.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		Path string `json:"path" jsonschema:"the document, as source_list gives it"`
	}) (*sdk.CallToolResult, struct {
		Started bool   `json:"started"`
		Says    string `json:"says"`
	}, error) {
		type out = struct {
			Started bool   `json:"started"`
			Says    string `json:"says"`
		}
		if core.Recognise == nil {
			return nil, out{}, fmt.Errorf("this installation cannot read scans")
		}

		// Nothing here happens inside this question. Fetching the models is
		// minutes and reading a book is an hour, and how far either has got is
		// among everything else the window shows being done.
		if !core.Recognise.Start(core.Vault, in.Path) {
			// Nothing here remembers a request that was not taken.
			return nil, out{Says: "another document is being read and this one was not taken; " +
				"nothing is reading it — ask again once source_list says none is being read"}, nil
		}
		says := "started; it runs in the background, the pages it has read are searchable " +
			"as it goes, and the window shows how far it has got"
		if !core.Recognise.Ready() {
			says = "started; what is needed to read scans is being fetched first, about 160 MB"
		}
		return nil, out{Started: true, Says: says}, nil
	})
}

// Recognising is what the tools need in order to read a document: a way to
// begin, a way to say how far it has got, and whether it could begin at once.
//
// It is an interface so that a server can be built without one, and so that the
// tools can say "it has started" rather than "there is nothing to read with".
type Recognising interface {
	// Ready says whether reading could begin now without waiting for anything
	// to arrive.
	Ready() bool
	// Running says whether a document is being read.
	Running() bool
	// Start begins reading one document behind whoever asked, and says whether
	// it began. It does not begin a second while one runs, and it runs under
	// the application rather than under the call that asked for it.
	Start(v domain.Vault, path string) bool
}
