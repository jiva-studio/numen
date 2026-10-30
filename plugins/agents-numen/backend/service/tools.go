package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
	"net/http"
	neturl "net/url"
	"slices"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// errPlainToken is a tool server the token would reach unencrypted.
var errPlainToken = errors.New("the vault's tools are reached only over https, or over http on this machine")

// emptySchema is the parameters of a tool whose server declares none.
var emptySchema = json.RawMessage(`{"type":"object","properties":{}}`)

// toolbox is the vault's tool server as seen by one run, and the tools of it the
// run may call. A toolbox with no session offers nothing.
type toolbox struct {
	session *mcp.ClientSession
	tools   []functionTool
	// meta is the context of the task, sent as the `_meta` of every call.
	meta mcp.Meta
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

// openToolbox connects to the tool server of the task and lists the tools of it the task names. An
// empty url, or a task naming no tool, is a run with no tools, and no connection is made.
func openToolbox(ctx context.Context, task *agentrun.AskAgentRequest) (*toolbox, error) {
	url, named := task.GetToolsUrl(), task.GetTools()
	if url == "" || len(named) == 0 {
		return &toolbox{session: nil, tools: nil, meta: nil}, nil
	}
	if address, err := neturl.Parse(url); err != nil || !isPrivate(address) {
		return nil, errPlainToken
	}
	transport := &mcp.StreamableClientTransport{
		Endpoint: url,
		HTTPClient: &http.Client{
			Transport:     bearer{token: task.GetToolsToken(), base: http.DefaultTransport},
			CheckRedirect: refuseDowngrade,
			Jar:           nil,
			Timeout:       0,
		},
		MaxRetries:           0,
		DisableStandaloneSSE: true,
		OAuthHandler:         nil,
		MaxEventSize:         0,
	}
	client := mcp.NewClient(&mcp.Implementation{
		Name: "numen-agent", Title: "", Description: "", Version: "1.0.0", WebsiteURL: "", Icons: nil,
	}, nil)
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("connect to the vault's tools: %w", err)
	}

	box := &toolbox{session: session, tools: nil, meta: readMeta(task.GetContext())}
	for tool, err := range session.Tools(ctx, nil) {
		if err != nil {
			return nil, errors.Join(fmt.Errorf("list the vault's tools: %w", err), session.Close())
		}
		if slices.Contains(named, tool.Name) {
			box.tools = append(box.tools, readFunctionTool(tool))
		}
	}
	return box, nil
}

// readFunctionTool is the tool as the chat API is offered it.
func readFunctionTool(tool *mcp.Tool) functionTool {
	var proposed functionTool
	proposed.Type = "function"
	proposed.Function.Name = tool.Name
	proposed.Function.Description = tool.Description
	proposed.Function.Parameters = emptySchema
	if tool.InputSchema != nil {
		if schema, err := json.Marshal(tool.InputSchema); err == nil {
			proposed.Function.Parameters = schema
		}
	}
	return proposed
}

// call runs the tool with the arguments the model wrote and returns the result
// for the model: the tool's text, or the error.
func (b *toolbox) call(ctx context.Context, name, arguments string) string {
	if b.session == nil || !slices.ContainsFunc(b.tools, func(t functionTool) bool { return t.Function.Name == name }) {
		return fmt.Sprintf("error: there is no tool %q", name)
	}
	args := json.RawMessage(arguments)
	if strings.TrimSpace(arguments) == "" {
		args = json.RawMessage(`{}`)
	}
	if !json.Valid(args) {
		return "error: the arguments are not JSON"
	}
	result, err := b.session.CallTool(ctx, &mcp.CallToolParams{
		Meta: b.meta, Name: name, Arguments: args, InputResponses: nil, RequestState: "",
	})
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

// close ends the session with the tool server.
func (b *toolbox) close() error {
	if b.session == nil {
		return nil
	}
	return b.session.Close()
}

// readMeta is the context of a task as MCP `_meta`, nil for a task with none.
func readMeta(values map[string]string) mcp.Meta {
	if len(values) == 0 {
		return nil
	}
	meta := make(mcp.Meta, len(values))
	for key, value := range values {
		meta[key] = value
	}
	return meta
}
