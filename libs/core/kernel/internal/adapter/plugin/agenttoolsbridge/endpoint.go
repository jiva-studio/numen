package agenttoolsbridge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/localendpoint"
)

// shutdownLimit is how long the calls in flight have to finish when the endpoint closes. A connection
// with a request in progress is cut after it, and close reports the cut.
const shutdownLimit = 2 * time.Second

// shutdownPoll is how often close looks for a connection with a request in progress.
const shutdownPoll = 10 * time.Millisecond

// endpointPath is where the endpoint answers MCP over Streamable HTTP.
const endpointPath = "/mcp"

// endpoint is an MCP server on a loopback port that serves a fixed set of tools behind a token.
type endpoint struct {
	url      string
	token    string
	server   *http.Server
	stopped  chan struct{}
	limit    time.Duration
	statesMu sync.Mutex
	states   map[net.Conn]http.ConnState
	once     sync.Once
	err      error
	serveErr error
}

func (e *endpoint) trackConnection(conn net.Conn, state http.ConnState) {
	e.statesMu.Lock()
	defer e.statesMu.Unlock()
	if state == http.StateClosed || state == http.StateHijacked {
		delete(e.states, conn)
		return
	}
	e.states[conn] = state
}

func (e *endpoint) getActiveCount() int {
	e.statesMu.Lock()
	defer e.statesMu.Unlock()
	count := 0
	for _, state := range e.states {
		if state == http.StateActive {
			count++
		}
	}
	return count
}

// toolCaller runs a tool on the plugin that listed it.
type toolCaller interface {
	callTool(ctx context.Context, one tool, arguments []byte, scope callScope) (*v1.CallToolResponse, error)
}

func serveEndpoint(ctx context.Context, tools []tool, scope callScope, caller toolCaller) (*endpoint, error) {
	token, err := localendpoint.NewToken()
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

	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen for the agent's tools: %w", err)
	}
	mux := http.NewServeMux()
	mux.Handle(endpointPath, sdk.NewStreamableHTTPHandler(func(*http.Request) *sdk.Server { return server }, nil))
	served := &endpoint{
		url:     (&url.URL{Scheme: "http", Host: listener.Addr().String(), Path: endpointPath}).String(),
		token:   token,
		server:  &http.Server{Handler: requireToken(token, mux), ReadHeaderTimeout: 10 * time.Second},
		stopped: make(chan struct{}),
		limit:   shutdownLimit,
		states:  make(map[net.Conn]http.ConnState),
	}
	served.server.ConnState = served.trackConnection
	go func() {
		defer close(served.stopped)
		// Serve returns http.ErrServerClosed once close is called, which is how it is meant to end.
		if err := served.server.Serve(listener); !errors.Is(err, http.ErrServerClosed) {
			served.serveErr = err
		}
	}()
	return served, nil
}

func (e *endpoint) close(ctx context.Context) error {
	e.once.Do(func() {
		ctx, cancel := context.WithTimeout(ctx, e.limit)
		defer cancel()
		e.err = e.shutdown(ctx)
		<-e.stopped
		e.err = errors.Join(e.err, e.serveErr)
	})
	return e.err
}

func (e *endpoint) shutdown(ctx context.Context) error {
	result := make(chan error, 1)
	go func() { result <- e.server.Shutdown(ctx) }()
	poll := time.NewTicker(shutdownPoll)
	defer poll.Stop()
	for {
		select {
		case err := <-result:
			if err != nil {
				return errors.Join(err, e.server.Close())
			}
			return nil
		case <-poll.C:
			if e.getActiveCount() == 0 {
				return errors.Join(e.server.Close(), <-result)
			}
		}
	}
}

func declareTool(spec *v1.PluginTool) *sdk.Tool {
	hints := spec.GetHints()
	isDestructive, isOpenWorld := hints.GetDestructive(), hints.GetOpenWorld()
	declared := &sdk.Tool{
		Name:        spec.GetName(),
		Title:       spec.GetTitle(),
		Description: spec.GetDescription(),
		InputSchema: readInputSchema(spec.GetInputSchema()),
		Annotations: &sdk.ToolAnnotations{
			ReadOnlyHint:    hints.GetReadOnly(),
			DestructiveHint: &isDestructive,
			IdempotentHint:  hints.GetIdempotent(),
			OpenWorldHint:   &isOpenWorld,
		},
	}
	if schema, isObject := readObject(spec.GetOutputSchema()); isObject && hasObjectType(schema) {
		declared.OutputSchema = json.RawMessage(spec.GetOutputSchema())
	}
	return declared
}

func readInputSchema(raw []byte) json.RawMessage {
	schema, isObject := readObject(raw)
	if !isObject {
		return json.RawMessage(`{"type":"object"}`)
	}
	if hasObjectType(schema) {
		return json.RawMessage(raw)
	}
	schema["type"] = json.RawMessage(`"object"`)
	encoded, err := json.Marshal(schema)
	if err != nil {
		return json.RawMessage(`{"type":"object"}`)
	}
	return encoded
}

func readObject(raw []byte) (map[string]json.RawMessage, bool) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil || object == nil {
		return nil, false
	}
	return object, true
}

func hasObjectType(schema map[string]json.RawMessage) bool {
	var schemaType string
	return json.Unmarshal(schema["type"], &schemaType) == nil && schemaType == "object"
}

func readResult(name string, res *v1.CallToolResponse) *sdk.CallToolResult {
	switch res.GetError() {
	case v1.ToolErrorCode_TOOL_ERROR_CODE_UNSPECIFIED:
	case v1.ToolErrorCode_TOOL_ERROR_CODE_UNKNOWN_TOOL:
		return newErrorResult("unknown tool: " + name)
	case v1.ToolErrorCode_TOOL_ERROR_CODE_BAD_ARGUMENTS:
		return newErrorResult("the arguments are not a JSON object")
	}
	answer := res.GetResult()
	result := &sdk.CallToolResult{IsError: answer.GetIsError()}
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

func requireToken(token string, next http.Handler) http.Handler {
	authenticated := localendpoint.RequireBearerToken(token, next)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origins, hasOrigin := r.Header["Origin"]; hasOrigin && (len(origins) != 1 || !isLoopbackOrigin(origins[0])) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		authenticated.ServeHTTP(w, r)
	})
}

func isLoopbackOrigin(origin string) bool {
	parsed, err := url.Parse(origin)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return false
	}
	host := parsed.Hostname()
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
