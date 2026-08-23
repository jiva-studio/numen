package mcp

import (
	"context"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/agent"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Words are how a tool is spoken about to a person: what it is called, and
// which of its arguments says what a call was about.
//
// Both are the tool's own declaration. A display name is the title it carries,
// falling back to the name it is served under; what a call is about is the
// first argument the tool requires.
type Words struct {
	Title string
	About string
	// Inside names the field of one element that says which element it is, for
	// a call that takes a collection. Creating three notes is about three
	// titles, and the collection itself is about nothing a person can read.
	Inside string
	// Kind is what this call does to the vault.
	Kind agent.Kind
	// Stood and Becomes name the arguments carrying the text a call replaces
	// and what it puts in that text's place. Both are empty for a call that
	// replaces no stretch.
	Stood   string
	Becomes string
}

// doing is what each tool this vault serves does, and the arguments a call
// naming a stretch of a note carries it in.
//
// The tools are written out by hand and so is this. A schema says what a call
// takes and cannot say what taking it means.
var doing = map[string]Words{
	"note_search":        {Kind: agent.Search},
	"note_get":           {Kind: agent.Read},
	"note_read":          {Kind: agent.Read},
	"note_neighbourhood": {Kind: agent.Read},
	"note_create":        {Kind: agent.Edit},
	"note_write":         {Kind: agent.Edit},
	"note_edit":          {Kind: agent.Edit, Stood: "stood", Becomes: "becomes"},
	"note_rename":        {Kind: agent.Move},
	"note_move":          {Kind: agent.Move},
	"note_remove":        {Kind: agent.Remove},
	"note_focus":         {Kind: agent.Read},
	"link_add":           {Kind: agent.Edit},
	"link_update":        {Kind: agent.Edit},
	"link_remove":        {Kind: agent.Edit},
	"link_list":          {Kind: agent.Read},
	"vault_get":          {Kind: agent.Read},
	"vault_named":        {Kind: agent.Read},
	"vault_problems":     {Kind: agent.Read},
	"source_list":        {Kind: agent.Read},
	"source_read":        {Kind: agent.Read},
	"source_show":        {Kind: agent.Read},
	// Reading a document changes what the vault holds — it writes down what a
	// model saw — so it is shown as a change and not as a look.
	"source_recognise": {Kind: agent.Edit},
}

// Vocabulary asks the server what it serves, and reads the answer.
//
// What the window says about a call is what an agent was told about it.
func Vocabulary(ctx context.Context, core Core) (map[string]Words, error) {
	server := New(core)
	here, there := sdk.NewInMemoryTransports()
	if _, err := server.Connect(ctx, here, nil); err != nil {
		return nil, err
	}
	client := sdk.NewClient(&sdk.Implementation{Name: "numen", Version: Version}, nil)
	session, err := client.Connect(ctx, there, nil)
	if err != nil {
		return nil, err
	}
	defer session.Close()

	listed, err := session.ListTools(ctx, nil)
	if err != nil {
		return nil, err
	}

	words := make(map[string]Words, len(listed.Tools))
	for _, tool := range listed.Tools {
		about := firstRequired(tool.InputSchema)
		words[tool.Name] = Words{
			Title:   titleOf(tool),
			About:   about,
			Inside:  firstRequiredInside(tool.InputSchema, about),
			Kind:    doing[tool.Name].Kind,
			Stood:   doing[tool.Name].Stood,
			Becomes: doing[tool.Name].Becomes,
		}
	}
	return words, nil
}

// titleOf is the name to show, in the order the protocol gives it.
func titleOf(tool *sdk.Tool) string {
	if tool.Title != "" {
		return tool.Title
	}
	if tool.Annotations != nil && tool.Annotations.Title != "" {
		return tool.Annotations.Title
	}
	return tool.Name
}

// firstRequired is the argument a call cannot be made without, and so the one
// that says what it was about. A tool that requires nothing is about nothing.
func firstRequired(schema any) string {
	declared, ok := schema.(map[string]any)
	if !ok {
		return ""
	}
	required, ok := declared["required"].([]any)
	if !ok || len(required) == 0 {
		return ""
	}
	first, _ := required[0].(string)
	return first
}

// holds reports whether a declared type is the one named. A field that may be
// left out is declared as several types at once, so the declaration is a name or
// a list of them.
func holds(declared any, kind string) bool {
	switch value := declared.(type) {
	case string:
		return value == kind
	case []any:
		for _, one := range value {
			if name, _ := one.(string); name == kind {
				return true
			}
		}
	}
	return false
}

// firstRequiredInside is what one element of a collection argument is named by:
// the field that element cannot be given without. Empty for an argument that is
// not a collection of things with names of their own.
func firstRequiredInside(schema any, field string) string {
	declared, ok := schema.(map[string]any)
	if !ok || field == "" {
		return ""
	}
	properties, ok := declared["properties"].(map[string]any)
	if !ok {
		return ""
	}
	argument, ok := properties[field].(map[string]any)
	if !ok {
		return ""
	}
	if !holds(argument["type"], "array") {
		return ""
	}
	return firstRequired(argument["items"])
}
