package mcp

import (
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// A Source is one source of a vault that is not a note, as an agent is told
// about it.
type Source struct {
	Path string `json:"path"`
	// ShouldRead says whether this document relies on what a model read in it. A
	// recognition still running relies on the pages it has reached, so this is true
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
