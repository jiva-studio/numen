package claudecode

import "github.com/jiva-studio/numen/modules/libs/core/port"

// Name is what the server this agent is served by calls itself, and the prefix
// its tools arrive under.
const Name = "numen"

// prefix is what a tool of this vault's server is called under once it reaches
// an agent.
const prefix = "mcp__" + Name + "__"

// Tool is what a tool of this vault's server is called once it reaches an
// agent.
func Tool(name string) string { return prefix + name }

// ToolDeclaration is how one tool is spoken about to a person: what it calls
// itself and what a call of it does to the vault. Both are the tool's own
// declaration, read from what the server serves.
type ToolDeclaration struct {
	Title string
	// Kind is what a call of this tool does to the vault. A tool that declares
	// nothing about it is port.StepToolCall.
	Kind port.StepKind
	// Arguments is how a call of it is read while it is being written.
	Arguments Arguments
}

// Arguments are the names this tool's own arguments arrive under. Nothing here
// is shown to anybody: they are what a call half written is read for the value
// that is.
type Arguments struct {
	// About names the argument that says what a call was about.
	About string
	// Element names the field of one element that says which element it is, for
	// a call that takes a collection.
	Element string
	// Match and Text name the arguments carrying the text a call replaces
	// and what it puts in that text's place. Both are empty for a call that
	// replaces no stretch.
	Match string
	Text  string
}

// notePath is the argument a tool of this vault names one note by. A call read
// by it is about a path, and that path is where the call is working.
const notePath = "path"

// spanStart and spanLength are the arguments a tool of this vault names a
// stretch of a source's text by.
const (
	spanStart  = "start"
	spanLength = "length"
)
