package mcp

import (
	"context"
	"fmt"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
)

// A Source is one source of a vault that is not a note, as an agent is told
// about it.
type Source struct {
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
// An agent asked to read a document finds out first which have been read
// already, so the listing is what makes the reading usable.
func addSourceTools(server *sdk.Server, core Core) {
	addSourceReadingTools(server, core)
	addSourceWritingTools(server, core)
}

func addSourceReadingTools(server *sdk.Server, core Core) {
	sdk.AddTool(server, &sdk.Tool{
		Name:  "source_list",
		Title: "List the documents a vault holds",
		Description: "List the books, papers and scans filed in the vault beside its notes, " +
			"and say of each whether a model has read it. A scanned document carries no " +
			"text of its own until it is read, and nothing in the words of a document " +
			"that does carry text says whether that text is any good. Use this before " +
			"asking for one to be read.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, _ struct{}) (*sdk.CallToolResult, struct {
		Documents []Source `json:"documents"`
		Reading   string   `json:"reading,omitempty" jsonschema:"what stands in the way of having a scan read now, when anything does"`
	}, error) {
		type out = struct {
			Documents []Source `json:"documents"`
			Reading   string   `json:"reading,omitempty" jsonschema:"what stands in the way of having a scan read now, when anything does"`
		}
		if core.Sources.Queries == nil {
			return nil, out{}, fmt.Errorf("this vault's sources are not open")
		}
		shown := core.shown()
		known, err := core.Sources.Queries.Fingerprints(ctx, string(shown.Vault.ID), domain.KindBook)
		if err != nil {
			return nil, out{}, err
		}
		read, err := core.Sources.Queries.Recognised(ctx, string(shown.Vault.ID), domain.KindBook)
		if err != nil {
			return nil, out{}, err
		}
		stands := make(map[string]bool, len(read))
		for _, one := range read {
			stands[one.Path] = one.Producer != ""
		}
		documents := make([]Source, 0, len(known))
		for path := range known {
			documents = append(documents, Source{Path: path, Read: stands[path]})
		}
		reading := ""
		if core.Sources.Recognise != nil {
			switch {
			case core.Sources.Recognise.Running():
				reading = "one document is being read now"
			case !core.Sources.Recognise.Ready():
				reading = "what is needed to read scans is not here yet, and is fetched when one is asked for"
			}
		}
		return nil, out{Documents: documents, Reading: reading}, nil
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
		Size     int    `json:"size" jsonschema:"how long the document's whole text is, in bytes, so what stands on either side can be asked for"`
	}, error) {
		type out = struct {
			Text     string `json:"text"`
			Location string `json:"location,omitempty" jsonschema:"where the run begins in the document's own numbering"`
			Start    int    `json:"start" jsonschema:"where the run begins, which is what was asked for held within the text"`
			Length   int    `json:"length" jsonschema:"how long the run is"`
			Size     int    `json:"size" jsonschema:"how long the document's whole text is, in bytes, so what stands on either side can be asked for"`
		}
		if core.Sources.Queries == nil {
			return nil, out{}, fmt.Errorf("this vault's sources are not open")
		}
		res, err := source.Read{
			Readers:   core.Readers,
			Sources:   core.Sources.Queries,
			Derived:   core.Sources.Derived,
			Documents: core.Sources.Documents,
		}.Execute(ctx, core.shown().Vault, in.Path, in.Start, in.Length)
		if err != nil {
			return nil, out{}, err
		}
		return nil, out{
			Text:     res.Text,
			Location: res.Location,
			Start:    res.Start,
			Length:   res.Length,
			Size:     res.Whole,
		}, nil
	})
}

func addSourceWritingTools(server *sdk.Server, core Core) {
	sdk.AddTool(server, &sdk.Tool{
		Name:  "source_recognise",
		Title: "Read a scanned document",
		Description: "Have a model read one scanned document and write down what it says. " +
			"The pages it has read are searchable as it goes, so a search finds the " +
			"beginning of a book long before the end of it is read; the part not yet " +
			"read answers nothing until it is. This is slow — an hour for a book — and " +
			"it is never done on its own, because whether a document's own text is any " +
			"good cannot be told from the text. Ask for it when a document is a scan, " +
			"or when what a search returns from one is nonsense. A document asked for " +
			"while another is being read waits its turn and is never refused, so this " +
			"is asked once and no more.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		Path string `json:"path" jsonschema:"the document, as source_list gives it"`
	}) (*sdk.CallToolResult, struct {
		Started bool   `json:"started" jsonschema:"whether the vault took this on, which it always does"`
		Doing   string `json:"doing" jsonschema:"what is happening now, in words to say back to the person"`
	}, error) {
		type out = struct {
			Started bool   `json:"started" jsonschema:"whether the vault took this on, which it always does"`
			Doing   string `json:"doing" jsonschema:"what is happening now, in words to say back to the person"`
		}
		if core.Sources.Recognise == nil {
			return nil, out{}, fmt.Errorf("this installation cannot read scans")
		}

		// Nothing here happens inside this question. Fetching the models is
		// minutes and reading a book is an hour, and how far either has got is
		// among everything else the window shows being done.
		doing := "started; it runs in the background, the pages it has read are searchable " +
			"as it goes, and the window shows how far it has got"
		switch core.Sources.Recognise.Start(core.shown().Vault, in.Path) {
		case port.Queued:
			doing = "queued; another document is being read and this one is in line behind " +
				"it — nothing more is needed, it begins when that reading is over"
		default:
			if !core.Sources.Recognise.Ready() {
				doing = "started; what is needed to read scans is being fetched first, about 160 MB"
			}
		}
		return nil, out{Started: true, Doing: doing}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:  "source_transcribe",
		Title: "Write down what a recording says",
		Description: "Have a model listen to one recording and write down the words it " +
			"carries. What has been heard is searchable as it goes, so a search finds " +
			"the first minutes of a talk long before the last of them are heard. This " +
			"is slow — about as long as the recording itself. A vault's recordings are " +
			"listened to on their own where the installation is set to; ask for this " +
			"when one is wanted now, or when the installation leaves it to the hand. A " +
			"recording asked for by name is heard whatever the installation listens to " +
			"on its own, and waits behind nothing but the recordings asked for before " +
			"it, so this is asked once and no more.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		Path string `json:"path" jsonschema:"the recording, as source_list gives it"`
	}) (*sdk.CallToolResult, struct {
		Started bool   `json:"started" jsonschema:"whether the vault took this on, which it always does"`
		Doing   string `json:"doing" jsonschema:"what is happening now, in words to say back to the person"`
	}, error) {
		type out = struct {
			Started bool   `json:"started" jsonschema:"whether the vault took this on, which it always does"`
			Doing   string `json:"doing" jsonschema:"what is happening now, in words to say back to the person"`
		}
		if core.Sources.Transcribe == nil {
			return nil, out{}, fmt.Errorf("this installation cannot hear recordings")
		}

		// Nothing here happens inside this question. Fetching the models is
		// minutes and transcribing a talk is an hour, and how far either has got
		// is among everything else the window shows being done.
		doing := "started; it runs in the background, and what has been transcribed is " +
			"searchable as it goes"
		switch core.Sources.Transcribe.Start(core.shown().Vault, in.Path) {
		case port.Queued:
			doing = "queued; another recording is being transcribed and this one is in line " +
				"behind it — nothing more is needed, it begins when that one is over"
		default:
			if !core.Sources.Transcribe.Ready() {
				doing = "started; what is needed to transcribe recordings is being fetched first"
			}
		}
		return nil, out{Started: true, Doing: doing}, nil
	})
}

// Recogniser is what the tools need in order to read a document: a way to
// begin, a way to say how far it has got, and whether it could begin at once.
//
// It is an interface so that a server can be built without one, and so that the
// tools can say "it has started" rather than "there is nothing to read with".
type Recogniser interface {
	// Ready says whether reading could begin now without waiting for anything
	// to arrive.
	Ready() bool
	// Running says whether a document is being read.
	Running() bool
	// Start reads one document behind whoever asked, and says whether it began
	// now or waits behind the reading already going. A document is never
	// refused, and it runs under the application rather than under the call
	// that asked for it.
	Start(v domain.Vault, path string) port.Taking
}

// Transcriber is what the tools need in order to hear a recording: a way to
// begin, and whether beginning would wait for anything to arrive.
//
// A server built without one serves no tool that would listen.
type Transcriber interface {
	// Ready says whether listening could begin now without waiting for anything
	// to arrive.
	Ready() bool
	// Start listens to one recording behind whoever asked, and says whether it
	// began now or waits behind the listening already going. A recording is
	// never refused, and it runs under the application rather than under the
	// call that asked for it.
	Start(v domain.Vault, path string) port.Taking
}
