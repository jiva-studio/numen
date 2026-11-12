// Package hosttools reaches the tools of the host: it lists the tools the permissions of the plugin allow
// and calls one with the context and the vault of the task.
package hosttools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/conversation"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/ports"
)

// emptySchema is the parameters of a tool that declares none.
const emptySchema = `{"type":"object","properties":{}}`

// Tools calls the tools of the host through a client.
type Tools struct {
	client tools.Client
}

var _ ports.Tools = (*Tools)(nil)

// New returns Tools that reach the host through client.
func New(client tools.Client) *Tools {
	return &Tools{client: client}
}

// FromEnvironment returns Tools of the host the process environment names.
func FromEnvironment() (*Tools, error) {
	endpoint, err := hostcall.FromEnv()
	if err != nil {
		return nil, err
	}
	return New(tools.NewClient(endpoint)), nil
}

// FromVariables returns Tools of the host the variables name.
func FromVariables(vars map[string]string) (*Tools, error) {
	endpoint, err := hostcall.FromVars(vars)
	if err != nil {
		return nil, err
	}
	return New(tools.NewClient(endpoint)), nil
}

// ListTools lists the tools of the host, in the order of the listing.
func (t *Tools) ListTools(ctx context.Context) ([]conversation.Tool, error) {
	listing, err := t.client.ListTools(ctx)
	if err != nil {
		return nil, fmt.Errorf("list the tools of the host: %w", err)
	}
	out := make([]conversation.Tool, 0, len(listing.Tools))
	for _, one := range listing.Tools {
		parameters := json.RawMessage(emptySchema)
		if schema := one.GetInputSchema(); len(schema) > 0 {
			parameters = schema
		}
		out = append(out, conversation.Tool{
			Name:         one.GetName(),
			Title:        one.GetTitle(),
			Description:  one.GetDescription(),
			Parameters:   parameters,
			IsReadOnly:   one.GetEffect() == tools.ReadOnly,
			Instructions: one.GetInstructions(),
		})
	}
	return out, nil
}

// CallTool calls the tool and returns its result as text. A call the host refuses or that fails is a
// result marked as an error, whose text the model reads.
func (t *Tools) CallTool(ctx context.Context, name, arguments string, scope conversation.Scope) conversation.ToolResult {
	args := []byte(arguments)
	if strings.TrimSpace(arguments) == "" {
		args = []byte(`{}`)
	}
	if !json.Valid(args) {
		return conversation.ToolResult{Text: "error: the arguments are not JSON", IsError: true}
	}
	response, err := t.client.InvokeTool(ctx, tools.Call{Name: name, Arguments: args, Context: scope.Context, VaultID: scope.VaultID})
	if err != nil {
		return conversation.ToolResult{Text: fmt.Sprintf("error: %v", err), IsError: true}
	}
	switch response.GetError() {
	case tools.ErrorCodeUnknownTool:
		return conversation.ToolResult{Text: fmt.Sprintf("error: there is no tool %q", name), IsError: true}
	case tools.ErrorCodeBadArguments:
		return conversation.ToolResult{Text: "error: the arguments do not fit the tool", IsError: true}
	case tools.ErrorCodeUnspecified:
	}
	result := response.GetResult()
	text := result.GetText()
	if text == "" && len(result.GetStructured()) > 0 {
		text = string(result.GetStructured())
	}
	if result.GetIsError() {
		return conversation.ToolResult{Text: "error: " + text, IsError: true}
	}
	return conversation.ToolResult{Text: text, IsError: false}
}
