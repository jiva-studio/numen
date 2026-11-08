package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	neturl "net/url"
	"slices"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
)

// errPlainToken is a tool server the token would reach unencrypted.
var errPlainToken = errors.New("the vault's tools are reached only over https, or over http on this machine")

// emptySchema is the parameters of a tool whose server declares none.
const emptySchema = `{"type":"object","properties":{}}`

// toolbox is the vault's tool server as seen by one run, and the tools of it the
// run may call. A toolbox with no session offers nothing.
type toolbox struct {
	session *sdk.ClientSession
	tools   []functionTool
	meta    sdk.Meta
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

func openToolbox(ctx context.Context, task *agentrun.Request) (*toolbox, error) {
	url, named := task.ToolsURL, task.Tools
	if url == "" || len(named) == 0 {
		return &toolbox{session: nil, tools: nil, meta: nil}, nil
	}
	if address, err := neturl.Parse(url); err != nil || !isPrivate(address) {
		return nil, errPlainToken
	}
	transport := &sdk.StreamableClientTransport{
		Endpoint: url,
		HTTPClient: &http.Client{
			Transport:     bearer{token: task.ToolsToken, base: http.DefaultTransport},
			CheckRedirect: refuseDowngrade,
			Jar:           nil,
			Timeout:       0,
		},
		MaxRetries:           0,
		DisableStandaloneSSE: true,
		OAuthHandler:         nil,
		MaxEventSize:         0,
	}
	client := sdk.NewClient(&sdk.Implementation{
		Name: "numen-agent", Title: "", Description: "", Version: "1.0.0", WebsiteURL: "", Icons: nil,
	}, nil)
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("connect to the vault's tools: %w", err)
	}

	box := &toolbox{session: session, tools: nil, meta: readMeta(task.Context)}
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

func readFunctionTool(tool *sdk.Tool) functionTool {
	var proposed functionTool
	proposed.Type = "function"
	proposed.Function.Name = tool.Name
	proposed.Function.Description = tool.Description
	proposed.Function.Parameters = json.RawMessage(emptySchema)
	if tool.InputSchema != nil {
		if schema, err := json.Marshal(tool.InputSchema); err == nil {
			proposed.Function.Parameters = schema
		}
	}
	return proposed
}

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
	result, err := b.session.CallTool(ctx, &sdk.CallToolParams{
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

func readResultText(result *sdk.CallToolResult) string {
	parts := make([]string, 0, len(result.Content))
	for _, content := range result.Content {
		if text, ok := content.(*sdk.TextContent); ok {
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

func (b *toolbox) close() error {
	if b.session == nil {
		return nil
	}
	return b.session.Close()
}

func readMeta(values map[string]string) sdk.Meta {
	if len(values) == 0 {
		return nil
	}
	meta := make(sdk.Meta, len(values))
	for key, value := range values {
		meta[key] = value
	}
	return meta
}
