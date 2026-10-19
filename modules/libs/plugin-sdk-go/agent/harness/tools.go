package harness

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// toolPrefix is what an agent program puts before the name of a tool of the vault's server.
const toolPrefix = "mcp__numen__"

// emptySchema is the parameters of a tool whose server declares none.
var emptySchema = json.RawMessage(`{"type":"object","properties":{}}`)

// toolbox is the vault's tool server as one run reaches it, and the tools of it the run may call.
// A toolbox with no session offers nothing.
type toolbox struct {
	session *mcp.ClientSession
	offered []functionTool
}

// bearer presents the token on every request to the tool server.
type bearer struct {
	token string
	base  http.RoundTripper
}

// RoundTrip sends the request with the token.
func (b bearer) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("Authorization", "Bearer "+b.token)
	return b.base.RoundTrip(req)
}

// openToolbox connects to the tool server at url and lists the tools of it the task names. An
// empty url is a run with no tools.
func openToolbox(ctx context.Context, url, token string, named []string) (*toolbox, error) {
	if url == "" {
		return &toolbox{}, nil
	}
	transport := &mcp.StreamableClientTransport{
		Endpoint:             url,
		HTTPClient:           &http.Client{Transport: bearer{token: token, base: http.DefaultTransport}},
		DisableStandaloneSSE: true,
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "numen-agent", Version: "1.0.0"}, nil)
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("connect to the vault's tools: %w", err)
	}

	allowed := make([]string, 0, len(named))
	for _, name := range named {
		allowed = append(allowed, strings.TrimPrefix(name, toolPrefix))
	}
	box := &toolbox{session: session}
	for tool, err := range session.Tools(ctx, nil) {
		if err != nil {
			_ = session.Close()
			return nil, fmt.Errorf("list the vault's tools: %w", err)
		}
		if slices.Contains(allowed, tool.Name) {
			box.offered = append(box.offered, readFunctionTool(tool))
		}
	}
	return box, nil
}

// readFunctionTool is the tool as the chat API is offered it.
func readFunctionTool(tool *mcp.Tool) functionTool {
	var offered functionTool
	offered.Type = "function"
	offered.Function.Name = tool.Name
	offered.Function.Description = tool.Description
	offered.Function.Parameters = emptySchema
	if tool.InputSchema != nil {
		if schema, err := json.Marshal(tool.InputSchema); err == nil {
			offered.Function.Parameters = schema
		}
	}
	return offered
}

// call runs the tool with the arguments the model wrote and returns what the model is told: the
// tool's text, or what went wrong.
func (b *toolbox) call(ctx context.Context, name, arguments string) string {
	if b.session == nil || !slices.ContainsFunc(b.offered, func(t functionTool) bool { return t.Function.Name == name }) {
		return fmt.Sprintf("error: there is no tool %q", name)
	}
	args := json.RawMessage(arguments)
	if strings.TrimSpace(arguments) == "" {
		args = json.RawMessage(`{}`)
	}
	if !json.Valid(args) {
		return "error: the arguments are not JSON"
	}
	result, err := b.session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	text := readResultText(result)
	if result.IsError {
		return "error: " + text
	}
	return text
}

// readResultText is the text of what a tool returned, each content written out on its own line.
func readResultText(result *mcp.CallToolResult) string {
	parts := make([]string, 0, len(result.Content))
	for _, content := range result.Content {
		if text, ok := content.(*mcp.TextContent); ok {
			parts = append(parts, text.Text)
			continue
		}
		if raw, err := json.Marshal(content); err == nil {
			parts = append(parts, string(raw))
		}
	}
	if len(parts) == 0 && result.StructuredContent != nil {
		if raw, err := json.Marshal(result.StructuredContent); err == nil {
			parts = append(parts, string(raw))
		}
	}
	return strings.Join(parts, "\n")
}

// close lets go of the tool server.
func (b *toolbox) close() {
	if b.session != nil {
		_ = b.session.Close()
	}
}
