// Package mcpendpoint serves the tools of a session to its agent program: an MCP server over Streamable
// HTTP on a loopback port, behind a token made for it. Any process on the machine can reach a loopback
// port, so the endpoint answers only a request that presents the token and comes from no page outside
// the machine.
package mcpendpoint

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
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/agents-codex/backend/usecase/task"
)

// shutdownLimit is how long the calls in flight have to finish when the endpoint closes. A connection
// with a request in progress is cut after it, and Close reports the cut.
const shutdownLimit = 2 * time.Second

// shutdownPoll is how often Close looks for a connection with a request in progress.
const shutdownPoll = 10 * time.Millisecond

// endpointPath is where the endpoint answers MCP.
const endpointPath = "/mcp"

// tokenBytes is the size of the token of an endpoint.
const tokenBytes = 32

// Server serves the tools of sessions, each on an endpoint of its own.
type Server struct{}

var _ task.ToolServer = Server{}

// Endpoint is an MCP server on a loopback port that serves a fixed set of tools behind a token. It answers
// 403 to a request from an origin outside the machine and 401 to one that does not present the token as a
// bearer credential.
type Endpoint struct {
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

// Serve starts an endpoint that serves the listed tools and runs each call with invoke. Every input schema
// is declared as an object schema; an output schema is declared only where the listing gives an object
// schema. A read-only tool is declared read-only and idempotent, a read-write one destructive, and one
// that requires link:open as reaching the open world.
func (Server) Serve(ctx context.Context, listed []*tools.ToolInfo, invoke task.Invoke) (task.ServedTools, error) {
	return serve(ctx, listed, invoke)
}

func serve(ctx context.Context, listed []*tools.ToolInfo, invoke task.Invoke) (*Endpoint, error) {
	token, err := newToken()
	if err != nil {
		return nil, err
	}
	server := sdk.NewServer(&sdk.Implementation{Name: "numen", Title: "", Description: "", Version: "1.0.0", WebsiteURL: "", Icons: nil}, nil)
	for _, one := range listed {
		name := one.GetName()
		server.AddTool(declareTool(one), func(ctx context.Context, req *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
			var arguments json.RawMessage
			if req.Params != nil {
				arguments = req.Params.Arguments
			}
			res, err := invoke(ctx, name, arguments)
			if err != nil {
				return newErrorResult(err.Error()), nil
			}
			return readResult(name, res), nil
		})
	}

	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen for the tools of the agent: %w", err)
	}
	mux := http.NewServeMux()
	mux.Handle(endpointPath, sdk.NewStreamableHTTPHandler(func(*http.Request) *sdk.Server { return server }, nil))
	served := &Endpoint{
		url:      (&url.URL{Scheme: "http", Host: listener.Addr().String(), Path: endpointPath}).String(),
		token:    token,
		server:   &http.Server{Handler: requireToken(token, mux), ReadHeaderTimeout: 10 * time.Second},
		stopped:  make(chan struct{}),
		limit:    shutdownLimit,
		statesMu: sync.Mutex{},
		states:   make(map[net.Conn]http.ConnState),
		once:     sync.Once{},
		err:      nil,
		serveErr: nil,
	}
	served.server.ConnState = served.trackConnection
	go func() {
		defer close(served.stopped)
		// Serve returns http.ErrServerClosed once Close is called, which is how it is meant to end.
		if err := served.server.Serve(listener); !errors.Is(err, http.ErrServerClosed) {
			served.serveErr = err
		}
	}()
	return served, nil
}

// GetURL returns the http address on 127.0.0.1 and the path /mcp where the endpoint answers.
func (e *Endpoint) GetURL() string { return e.url }

// GetToken returns the token a request presents as a bearer credential.
func (e *Endpoint) GetToken() string { return e.token }

// Close stops the endpoint once the calls in flight have finished or the shutdown limit has passed.
func (e *Endpoint) Close(ctx context.Context) error {
	e.once.Do(func() {
		ctx, cancel := context.WithTimeout(ctx, e.limit)
		defer cancel()
		e.err = e.shutdown(ctx)
		<-e.stopped
		e.err = errors.Join(e.err, e.serveErr)
	})
	return e.err
}

func (e *Endpoint) trackConnection(conn net.Conn, state http.ConnState) {
	e.statesMu.Lock()
	defer e.statesMu.Unlock()
	if state == http.StateClosed || state == http.StateHijacked {
		delete(e.states, conn)
		return
	}
	e.states[conn] = state
}

func (e *Endpoint) getActiveCount() int {
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

func (e *Endpoint) shutdown(ctx context.Context) error {
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

func declareTool(listed *tools.ToolInfo) *sdk.Tool {
	isDestructive := listed.GetEffect() == tools.ReadWrite
	isOpenWorld := slices.Contains(listed.GetRequires(), host.PermissionLinkOpen)
	declared := &sdk.Tool{
		Meta:         nil,
		Name:         listed.GetName(),
		Title:        listed.GetTitle(),
		Description:  listed.GetDescription(),
		InputSchema:  readInputSchema(listed.GetInputSchema()),
		OutputSchema: nil,
		Icons:        nil,
		Annotations: &sdk.ToolAnnotations{
			ReadOnlyHint:    listed.GetEffect() == tools.ReadOnly,
			DestructiveHint: &isDestructive,
			IdempotentHint:  listed.GetEffect() == tools.ReadOnly,
			OpenWorldHint:   &isOpenWorld,
			Title:           "",
		},
	}
	if schema, isObject := readObject(listed.GetOutputSchema()); isObject && hasObjectType(schema) {
		declared.OutputSchema = json.RawMessage(listed.GetOutputSchema())
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

func readResult(name string, res *tools.CallResponse) *sdk.CallToolResult {
	switch res.GetError() {
	case tools.ErrorCodeUnspecified:
	case tools.ErrorCodeUnknownTool:
		return newErrorResult("unknown tool: " + name)
	case tools.ErrorCodeBadArguments:
		return newErrorResult("the arguments are not a JSON object")
	}
	answer := res.GetResult()
	result := &sdk.CallToolResult{
		Meta: nil, Content: nil, StructuredContent: nil, IsError: answer.GetIsError(), InputRequests: nil, RequestState: "",
	}
	if structured := answer.GetStructured(); len(structured) > 0 {
		result.StructuredContent = json.RawMessage(structured)
	}
	text := answer.GetText()
	if text == "" {
		text = string(answer.GetStructured())
	}
	if text != "" {
		result.Content = []sdk.Content{newTextContent(text)}
	}
	return result
}

func newErrorResult(text string) *sdk.CallToolResult {
	return &sdk.CallToolResult{
		Meta: nil, Content: []sdk.Content{newTextContent(text)}, StructuredContent: nil, IsError: true, InputRequests: nil, RequestState: "",
	}
}

func newTextContent(text string) *sdk.TextContent {
	return &sdk.TextContent{Text: text, Meta: nil, Annotations: nil}
}

func newToken() (string, error) {
	raw := make([]byte, tokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("create a token for the tools of the agent: %w", err)
	}
	return hex.EncodeToString(raw), nil
}

func requireToken(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origins, hasOrigin := r.Header["Origin"]; hasOrigin && (len(origins) != 1 || !isLoopbackOrigin(origins[0])) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		presented, isBearer := strings.CutPrefix(strings.TrimSpace(r.Header.Get("Authorization")), "Bearer ")
		if !isBearer || subtle.ConstantTimeCompare([]byte(token), []byte(presented)) != 1 {
			w.Header().Set("WWW-Authenticate", `Bearer realm="numen"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func isLoopbackOrigin(origin string) bool {
	parsed, err := url.Parse(origin)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return false
	}
	hostname := parsed.Hostname()
	if hostname == "localhost" {
		return true
	}
	ip := net.ParseIP(hostname)
	return ip != nil && ip.IsLoopback()
}
