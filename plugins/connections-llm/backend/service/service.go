package service

import (
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
)

const (
	pluginID = "md.numen.connections.llm"

	typeOpenRouter = "openrouter"
	typeOpenAI     = "openai"
	typeCompatible = "openai-compatible"

	operationComplete = "complete"
	operationChat     = "chat"
	operationEmbed    = "embed"
	operationList     = "list"

	schemeHTTP = "http"
	poolingCLS = "cls"

	settingRequestTimeout = "requestTimeout"
	fieldBaseURL          = "baseUrl"
	secretAPIKey          = "apiKey"

	defaultCompleteLimit = 120 * time.Second
	defaultOtherLimit    = 60 * time.Second
	// firstRequestLimit bounds the first request to an OpenAI-compatible model: a local server loads
	// the model while it answers.
	firstRequestLimit = 10 * time.Minute
)

// Service carries out the operations of the connection types of the plugin. It sends a request only
// to an http or https address with a host and no credentials. A key goes over http only to a loopback
// address. Link-local, unspecified, multicast and cloud metadata addresses are rejected when the
// address is read and again when the host name is resolved. Redirects are not followed.
type Service struct {
	mu      sync.Mutex
	timeout time.Duration
	served  map[string]struct{}
	clients clients
}

// New creates a service with the default limits.
func New() *Service {
	return &Service{mu: sync.Mutex{}, timeout: 0, served: map[string]struct{}{}, clients: newClients()}
}

// Provider returns the roles the service fills.
func (s *Service) Provider() connection.Provider {
	return connection.Provider{Completer: s, Chatter: s, Embedder: s, Models: s, Profiles: s}
}

// Mount serves the provider on the plugin server and takes the settings of the plugin from Configure.
func Mount(srv *server.Server) error {
	svc := New()
	srv.SetUpdateConfig(func(values map[string]string, _ server.Secrets) { svc.Configure(values) })
	srv.OnClose(func() error {
		svc.clients.close()
		return nil
	})
	connection.Mount(srv, svc.Provider())
	return nil
}

// Configure reads the settings of the plugin. A request timeout that is not a positive number in
// seconds leaves the default of the operation.
func (s *Service) Configure(values map[string]string) {
	seconds, err := strconv.ParseFloat(strings.TrimSpace(values[settingRequestTimeout]), 64)
	var timeout time.Duration
	if err == nil && seconds > 0 && !math.IsInf(seconds, 0) && seconds < float64(math.MaxInt64/int64(time.Second)) {
		timeout = time.Duration(seconds * float64(time.Second))
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.timeout = timeout
}

func (s *Service) getLimit(call connection.Call, operation string) time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	if call.Type == typeCompatible {
		if _, isServed := s.served[servedKey(call)]; !isServed {
			return firstRequestLimit
		}
	}
	if s.timeout > 0 {
		return s.timeout
	}
	if operation == operationComplete {
		return defaultCompleteLimit
	}
	return defaultOtherLimit
}

func (s *Service) markServed(call connection.Call) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.served[servedKey(call)] = struct{}{}
}

func servedKey(call connection.Call) string {
	return call.Fields[fieldBaseURL] + "\x00" + call.Model
}
