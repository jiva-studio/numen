package agenttoolsbridge

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
)

// shutdownLimit is how long the calls in flight have to finish when the endpoint closes. A session
// left open holds its connection until it is cut.
const shutdownLimit = 2 * time.Second

// endpointPath is where the endpoint answers MCP over Streamable HTTP.
const endpointPath = "/mcp"

// endpoint is an MCP server on a loopback port that serves a fixed set of tools behind a token.
type endpoint struct {
	url    string
	token  string
	server *http.Server
	// stopped is closed once the server no longer serves.
	stopped chan struct{}
	once    sync.Once
	err     error
}

// toolCaller runs a tool on the plugin that listed it.
type toolCaller interface {
	callTool(ctx context.Context, one tool, arguments []byte, scope callScope) (*v1.CallToolResponse, error)
}

// serveEndpoint serves the tools on a loopback port the machine picks, under a new token. Every call
// is made in the scope given, whatever `_meta` the agent sends.
func serveEndpoint(tools []tool, scope callScope, caller toolCaller) (*endpoint, error) {
	token, err := createToken()
	if err != nil {
		return nil, err
	}
	server := sdk.NewServer(&sdk.Implementation{Name: "numen", Version: "1.0.0"}, nil)
	for _, one := range tools {
		server.AddTool(declareTool(one.spec), func(ctx context.Context, req *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
			var arguments json.RawMessage
			if req.Params != nil {
				arguments = req.Params.Arguments
			}
			res, err := caller.callTool(ctx, one, arguments, scope)
			if err != nil {
				return newErrorResult(err.Error()), nil
			}
			return readResult(one.spec.GetName(), res), nil
		})
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen for the agent's tools: %w", err)
	}
	mux := http.NewServeMux()
	mux.Handle(endpointPath, sdk.NewStreamableHTTPHandler(func(*http.Request) *sdk.Server { return server }, nil))
	served := &endpoint{
		url:     "http://" + listener.Addr().String() + endpointPath,
		token:   token,
		server:  &http.Server{Handler: requireToken(token, mux), ReadHeaderTimeout: 10 * time.Second},
		stopped: make(chan struct{}),
	}
	go func() {
		defer close(served.stopped)
		// Serve returns http.ErrServerClosed once close is called, which is how it is meant to end.
		_ = served.server.Serve(listener)
	}()
	return served, nil
}

// close stops the endpoint once and returns when it no longer serves. The calls in flight are given
// shutdownLimit, and the connections still open after it are cut.
func (e *endpoint) close(ctx context.Context) error {
	e.once.Do(func() {
		ctx, cancel := context.WithTimeout(ctx, shutdownLimit)
		defer cancel()
		if err := e.server.Shutdown(ctx); err != nil {
			e.err = errors.Join(err, e.server.Close())
		}
		<-e.stopped
	})
	return e.err
}

// declareTool is the tool as an MCP client is told of it.
func declareTool(spec *v1.PluginTool) *sdk.Tool {
	declared := &sdk.Tool{
		Name:        spec.GetName(),
		Title:       spec.GetTitle(),
		Description: spec.GetDescription(),
		InputSchema: readInputSchema(spec.GetInputSchema()),
		Annotations: &sdk.ToolAnnotations{ReadOnlyHint: spec.GetEffect() == v1.ToolEffect_TOOL_EFFECT_READ_ONLY},
	}
	if output := spec.GetOutputSchema(); json.Valid(output) && isObject(output) {
		declared.OutputSchema = json.RawMessage(output)
	}
	return declared
}

// readInputSchema is the input schema of a tool as a JSON object of type "object", which MCP
// requires. A schema that is no object is replaced by the empty one.
func readInputSchema(raw []byte) json.RawMessage {
	var schema map[string]any
	if err := json.Unmarshal(raw, &schema); err != nil || schema == nil {
		return json.RawMessage(`{"type":"object"}`)
	}
	if schema["type"] == "object" {
		return json.RawMessage(raw)
	}
	schema["type"] = "object"
	encoded, err := json.Marshal(schema)
	if err != nil {
		return json.RawMessage(`{"type":"object"}`)
	}
	return encoded
}

func isObject(raw []byte) bool {
	var object map[string]json.RawMessage
	return json.Unmarshal(raw, &object) == nil && object != nil
}

// readResult is what the plugin answered, as the result an agent reads.
func readResult(name string, res *v1.CallToolResponse) *sdk.CallToolResult {
	switch res.GetError() {
	case v1.ToolErrorCode_TOOL_ERROR_CODE_UNSPECIFIED:
	case v1.ToolErrorCode_TOOL_ERROR_CODE_UNKNOWN_TOOL:
		return newErrorResult("unknown tool: " + name)
	case v1.ToolErrorCode_TOOL_ERROR_CODE_BAD_ARGUMENTS:
		return newErrorResult("the arguments are not a JSON object")
	}
	answer := res.GetResult()
	result := &sdk.CallToolResult{IsError: answer.GetIsRefusal()}
	if structured := answer.GetStructured(); len(structured) > 0 {
		result.StructuredContent = json.RawMessage(structured)
	}
	text := answer.GetText()
	if text == "" {
		text = string(answer.GetStructured())
	}
	if text != "" {
		result.Content = []sdk.Content{&sdk.TextContent{Text: text}}
	}
	return result
}

func newErrorResult(text string) *sdk.CallToolResult {
	return &sdk.CallToolResult{IsError: true, Content: []sdk.Content{&sdk.TextContent{Text: text}}}
}

// createToken is 32 random bytes in hex.
func createToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("create the token of the agent's tools: %w", err)
	}
	return hex.EncodeToString(raw), nil
}

// requireToken answers only the requests that present the token as a bearer.
func requireToken(token string, next http.Handler) http.Handler {
	expected := []byte("Bearer " + token)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		presented := strings.TrimSpace(r.Header.Get("Authorization"))
		if subtle.ConstantTimeCompare([]byte(presented), expected) != 1 {
			w.Header().Set("WWW-Authenticate", `Bearer realm="numen"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
