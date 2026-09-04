package mcp

import (
	"context"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// Tool is how one tool is spoken about to a person: what it is called, and
// which of its arguments says what a call was about.
//
// Both are the tool's own declaration. A display name is the title it carries,
// falling back to the name it is served under; what a call is about is the
// first argument the tool requires.
type Tool struct {
	Title string
	About string
	// Inside names the field of one element that says which element it is, for
	// a call that takes a collection. Creating three notes is about three
	// titles, and the collection itself is about nothing a person can read.
	Inside string
	// Kind is what this call does to the vault.
	Kind port.StepKind
	// Match and Text name the arguments carrying the text a call replaces and
	// what it puts in that text's place. Both are empty for a call that
	// replaces no stretch.
	Match string
	Text  string
}

// doing is what each tool this vault serves does, and the arguments a call
// naming a stretch of a note carries it in.
//
// The tools are written out by hand and so is this. A schema says what a call
// takes and cannot say what taking it means.
var doing = map[string]Tool{
	"note_search":         {Kind: port.StepSearch},
	"note_titles":         {Kind: port.StepRead},
	"note_read":           {Kind: port.StepRead},
	"note_resolve":        {Kind: port.StepRead},
	"note_neighbourhood":  {Kind: port.StepRead},
	"note_create":         {Kind: port.StepEdit},
	"note_rewrite":        {Kind: port.StepEdit},
	"note_edit":           {Kind: port.StepEdit, Match: "match", Text: "text"},
	"note_rename":         {Kind: port.StepMove},
	"note_move":           {Kind: port.StepMove},
	"note_remove":         {Kind: port.StepRemove},
	"note_focus":          {Kind: port.StepRead},
	"file_read":           {Kind: port.StepRead},
	"card_stencil_list":   {Kind: port.StepRead},
	"card_read":           {Kind: port.StepRead},
	"card_showing":        {Kind: port.StepRead},
	"card_add":            {Kind: port.StepEdit},
	"card_edit":           {Kind: port.StepEdit},
	"card_value_remove":   {Kind: port.StepRemove},
	"card_remove":         {Kind: port.StepRemove},
	"card_section_add":    {Kind: port.StepEdit},
	"card_section_rename": {Kind: port.StepMove},
	"card_section_remove": {Kind: port.StepRemove},
	"card_deck_create":    {Kind: port.StepEdit},
	"card_stencil_create": {Kind: port.StepEdit},
	// A field's name stands in the stencil that declares it and in every card
	// that stencil cuts, so renaming it is a write to as many files as hold one.
	"card_field_rename": {Kind: port.StepMove},
	"link_add":          {Kind: port.StepEdit},
	"link_update":       {Kind: port.StepEdit},
	"link_remove":       {Kind: port.StepEdit},
	"link_list":         {Kind: port.StepRead},
	"window_tab_list":   {Kind: port.StepRead},
	"vault_get":         {Kind: port.StepRead},
	"vault_problems":    {Kind: port.StepRead},
	"vault_list":        {Kind: port.StepRead},
	"vault_add":         {Kind: port.StepEdit},
	"vault_rename":      {Kind: port.StepMove},
	"vault_forget":      {Kind: port.StepRemove},
	"vault_open":        {Kind: port.StepRead},
	"source_list":       {Kind: port.StepRead},
	"source_read":       {Kind: port.StepRead},
	"source_focus":      {Kind: port.StepRead},
	// Reading a document changes what the vault holds — it writes down what a
	// model saw — so it is shown as a change and not as a look. Listening to a
	// recording writes down what a model heard, and is shown the same way.
	"source_recognise":  {Kind: port.StepEdit},
	"source_transcribe": {Kind: port.StepEdit},
}

// Vocabulary asks the server what it serves, and reads the answer.
//
// What the window says about a call is what an agent was told about it.
func Vocabulary(ctx context.Context, core Core) (map[string]Tool, error) {
	return vocabulary(ctx, New(core))
}

// ReadingVocabulary is the same, for a window served the tools that read. A
// window is told about the tools it serves and no others.
func ReadingVocabulary(ctx context.Context, core Core) (map[string]Tool, error) {
	return vocabulary(ctx, NewReading(core))
}

// ReviewingVocabulary is the same, for the window a person runs their cards in.
func ReviewingVocabulary(ctx context.Context, core Core) (map[string]Tool, error) {
	return vocabulary(ctx, NewReviewing(core))
}

func vocabulary(ctx context.Context, server *sdk.Server) (map[string]Tool, error) {
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

	words := make(map[string]Tool, len(listed.Tools))
	for _, tool := range listed.Tools {
		about := firstRequired(tool.InputSchema)
		words[tool.Name] = Tool{
			Title:  titleOf(tool),
			About:  about,
			Inside: firstRequiredInside(tool.InputSchema, about),
			Kind:   doing[tool.Name].Kind,
			Match:  doing[tool.Name].Match,
			Text:   doing[tool.Name].Text,
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
