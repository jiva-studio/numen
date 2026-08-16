package mcp

import (
	"context"

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
		words[tool.Name] = Words{Title: titleOf(tool), About: firstRequired(tool.InputSchema)}
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
