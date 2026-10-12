package mcp

import (
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// A Source is one source of a vault that is not a note, as an agent is told
// about it.
type Source struct {
	Path string `json:"path"`
	// ShouldRead says whether this document stands on what a model read in it. A
	// recognition still running stands on the pages it has reached, so this is true
	// from the first of them.
	//
	// A scan that carries its own text says nothing about whether that text is
	// any good, so this is the only thing that can be said for certain.
	ShouldRead bool `json:"read"`
}

// addSourceTools adds the tools for the sources a vault holds beside its notes.
//
// An agent asked to read a document finds out first which have been read
// already, so the listing is what makes the reading usable.
func addSourceTools(server *sdk.Server, core Core) {
	addSourceReadTools(server, core)
	addSourceWriteTools(server, core)
}

// Recogniser is what the tools need in order to read a document: a way to
// begin, a way to say how far it has got, and whether it could begin at once.
//
// A server built without one serves the tool that would read anyway, and it
// answers that this installation cannot read scans.
type Recogniser interface {
	// Ready says whether recognition could begin now without waiting for anything
	// to arrive.
	Ready() bool
	// IsRunning says whether a document is being read.
	IsRunning() bool
	// Start reads one document behind whoever asked, and says whether it began
	// now or waits behind the recognition already going. A document is never
	// refused, and what it starts runs under the application.
	Start(v domain.Vault, path string) port.StartOutcome
}

// Transcriber is what the tools need in order to hear a recording: a way to
// begin, and whether beginning would wait for anything to arrive.
//
// A server built without one serves the tool that would listen anyway, and it
// answers that this installation cannot hear.
type Transcriber interface {
	// Ready says whether listening could begin now without waiting for anything
	// to arrive.
	Ready() bool
	// Start listens to one recording behind whoever asked, and says whether it
	// began now or waits behind the listening already going. A recording is
	// never refused, and what it starts runs under the application.
	Start(v domain.Vault, path string) port.StartOutcome
}
