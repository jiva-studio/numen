package mcp

import (
	"context"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

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
	Kind domain.StepKind
	// Match and Text name the arguments carrying the text a call replaces and
	// what it puts in that text's place. Both are empty for a call that
	// replaces no span.
	Match string
	Text  string
}

// doing is what each tool this vault serves does, and the arguments a call
// naming a span of a note carries it in.
//
// The tools are written out by hand and so is this. A schema says what a call
// takes and cannot say what taking it means.
var handler = map[string]Tool{
	"note_search":        {Kind: domain.StepSearch},
	"note_titles":        {Kind: domain.StepRead},
	"note_read":          {Kind: domain.StepRead},
	"note_resolve":       {Kind: domain.StepRead},
	"note_neighbourhood": {Kind: domain.StepRead},
	"note_create":        {Kind: domain.StepEdit},
	"note_import":        {Kind: domain.StepEdit},
	"note_rewrite":       {Kind: domain.StepEdit},
	"note_edit":          {Kind: domain.StepEdit, Match: "match", Text: "text"},
	"note_rename":        {Kind: domain.StepMove},
	"note_move":          {Kind: domain.StepMove},
	"note_remove":        {Kind: domain.StepRemove},
	"note_focus":         {Kind: domain.StepRead},
	"file_read":          {Kind: domain.StepRead},
	"link_add":           {Kind: domain.StepEdit},
	"link_update":        {Kind: domain.StepEdit},
	"link_remove":        {Kind: domain.StepEdit},
	"link_list":          {Kind: domain.StepRead},
	"window_tab_list":    {Kind: domain.StepRead},
	"vault_get":          {Kind: domain.StepRead},
	"vault_problems":     {Kind: domain.StepRead},
	"vault_list":         {Kind: domain.StepRead},
	"vault_add":          {Kind: domain.StepEdit},
	"vault_rename":       {Kind: domain.StepMove},
	"vault_remove":       {Kind: domain.StepRemove},
	"vault_open":         {Kind: domain.StepRead},
	"source_list":        {Kind: domain.StepRead},
	"source_read":        {Kind: domain.StepRead},
	"artifact_list":      {Kind: domain.StepRead},
	"artifact_read":      {Kind: domain.StepRead},
	"source_focus":       {Kind: domain.StepRead},
	// Reading a document changes what the vault holds — it writes down what a
	// model saw — so it is shown as a change and not as a look. Listening to a
	// recording writes down what a model heard, and is shown the same way.
	"source_recognise":  {Kind: domain.StepEdit},
	"source_transcribe": {Kind: domain.StepEdit},
}

// Vocabulary asks the server what it serves, and reads the answer.
//
// What the window says about a call is what an agent was told about it.
func Vocabulary(ctx context.Context, core Core) (map[string]Tool, error) {
	return vocabulary(ctx, newFull(ctx, core))
}

// ReadingVocabulary is the same, for a window served the tools that read. A
func ReadingVocabulary(ctx context.Context, core Core) (map[string]Tool, error) {
	return vocabulary(ctx, newReading(ctx, core))
}

// GetReviewVocabulary is the same, for the window a person runs their cards in.
func GetReviewVocabulary(ctx context.Context, core Core) (map[string]Tool, error) {
	return vocabulary(ctx, newReview(ctx, core))
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

	items, err := session.ListTools(ctx, nil)
	if err != nil {
		return nil, err
	}

	words := make(map[string]Tool, len(items.Tools))
	for _, tool := range items.Tools {
		about := firstRequired(tool.InputSchema)
		words[tool.Name] = Tool{
			Title:  titleOf(tool),
			About:  about,
			Inside: firstRequiredInside(tool.InputSchema, about),
			Kind:   getVocabularyKind(tool),
			Match:  handler[tool.Name].Match,
			Text:   handler[tool.Name].Text,
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

// isDeclaredAs reports whether a declared type is the one named. A field that
// may be left out is declared as several types at once, so the declaration is a
// name or a list of them.
func isDeclaredAs(declared any, kind string) bool {
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
	if !isDeclaredAs(argument["type"], "array") {
		return ""
	}
	return firstRequired(argument["items"])
}

// getVocabularyKind is how a call of the tool is shown: as the tool itself declares it where it is
// a plugin's, else as the core's own table has it.
func getVocabularyKind(tool *sdk.Tool) domain.StepKind {
	// A number that crossed a transport as JSON is a float.
	if declared, isDeclared := tool.Meta[stepMeta].(float64); isDeclared {
		return domain.StepKind(int(declared))
	}
	return handler[tool.Name].Kind
}
