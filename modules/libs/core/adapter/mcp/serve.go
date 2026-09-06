package mcp

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// DefaultAddr is where an agent looks when it is told nothing else. The port
// is fixed: it goes in a configuration file the person writes once.
const DefaultAddr = "127.0.0.1:7717"

// Endpoint is a running server: where it is, and what an agent must present to
// reach it.
type Endpoint struct {
	URL string

	server *http.Server
	// calls is the tool calls taken and not yet answered.
	calls calls
	// stop lets go of the goroutine waiting on the context, so that closing one
	// endpoint does not leave a watcher behind until the application exits.
	stop func()
	once sync.Once
}

// errClosed is what a call asked for after the door is shut gets.
var errClosed = errors.New("this vault is closing")

// counting takes what an agent asks for, so that closing can wait for what it
// is in the middle of.
//
// Every method an agent calls is bounded work — a tool call is a change to the
// vault and to the index behind it. The stream a session holds open is not a
// method and is not counted.
func (e *Endpoint) counting(next sdk.MethodHandler) sdk.MethodHandler {
	return func(ctx context.Context, method string, req sdk.Request) (sdk.Result, error) {
		if !e.calls.begin() {
			return nil, errClosed
		}
		defer e.calls.done()
		return next(ctx, method, req)
	}
}

// calls tracks the tool calls taken and not yet answered.
//
// What is counted is the call itself, and not the answer travelling back to
// the agent.
type calls struct {
	mu     sync.Mutex
	count  int
	sealed bool
	idle   chan struct{}
	over   bool
}

// begin takes a call, and refuses one that arrives after the door is shut.
func (c *calls) begin() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.sealed {
		return false
	}
	c.count++
	return true
}

func (c *calls) done() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.count--
	c.reckon()
}

// seal shuts the door on new calls and answers with what closes once the ones
// already taken have finished. The set it waits on is therefore finite and
// does not grow.
func (c *calls) seal() <-chan struct{} {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.sealed = true
	if c.idle == nil {
		c.idle = make(chan struct{})
	}
	c.reckon()
	return c.idle
}

// reckon ends the wait once nothing is running. The lock is held.
func (c *calls) reckon() {
	if !c.sealed || c.over || c.count > 0 {
		return
	}
	c.over = true
	close(c.idle)
}

// ServeHTTP starts the server and returns once it is listening.
//
// The listener is opened before returning, so a port already in use is an
// error the person sees at startup.
//
// Trouble, if it is given, is called with whatever stops the server later. It
// is how the person hears that it stopped.
func ServeHTTP(ctx context.Context, addr, token string, core Core, trouble func(error)) (*Endpoint, error) {
	return serve(ctx, addr, token, New(core), trouble)
}

// ServeReadingHTTP starts a server whose every tool reads. An agent answering
// through it changes nothing.
func ServeReadingHTTP(ctx context.Context, addr, token string, core Core, trouble func(error)) (*Endpoint, error) {
	return serve(ctx, addr, token, NewReading(core), trouble)
}

// ServeReviewingHTTP starts the server the window a person runs their cards in
// serves: everything that reads, and the cards of a deck.
func ServeReviewingHTTP(ctx context.Context, addr, token string, core Core, trouble func(error)) (*Endpoint, error) {
	return serve(ctx, addr, token, NewReviewing(core), trouble)
}

func serve(ctx context.Context, addr, token string, server *sdk.Server, trouble func(error)) (*Endpoint, error) {
	if addr == "" {
		addr = DefaultAddr
	}
	if token == "" {
		return nil, errors.New("a server an agent can reach needs a token to present")
	}

	endpoint := &Endpoint{}
	server.AddReceivingMiddleware(endpoint.counting)
	handler := sdk.NewStreamableHTTPHandler(
		func(*http.Request) *sdk.Server { return server },
		&sdk.StreamableHTTPOptions{
			// A page open in a browser can reach a port on this machine, and is
			// the one caller that arrives without being invited.
			CrossOriginProtection: &http.CrossOriginProtection{},
			SessionTimeout:        30 * time.Minute,
		},
	)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen on %s: %w", addr, err)
	}

	endpoint.URL = "http://" + listener.Addr().String() + "/mcp"
	endpoint.server = &http.Server{
		Handler:           behind(token, handler),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		err := endpoint.server.Serve(listener)
		if errors.Is(err, http.ErrServerClosed) || trouble == nil {
			return
		}
		trouble(err)
	}()
	stopped := make(chan struct{})
	endpoint.stop = func() { close(stopped) }
	go func() {
		select {
		case <-ctx.Done():
			// The context this was served under has ended, so the drain has no
			// time left: the listener and the idle connections go at once, and
			// the calls already running are still waited for.
			endpoint.Close(ctx)
		case <-stopped:
		}
	}()
	return endpoint, nil
}

// Close stops answering and waits for the calls it has taken.
//
// The transport is cut off first, within the bound the caller gives: an agent
// holding a session open is not a reason to keep the application running. What
// was already running is then waited for with no bound, so that by the time
// this answers nothing of an agent's is still changing the vault or the index
// behind it.
func (e *Endpoint) Close(ctx context.Context) error {
	if e == nil || e.server == nil {
		return nil
	}
	if e.stop != nil {
		e.once.Do(e.stop)
	}
	err := e.server.Shutdown(ctx)
	<-e.calls.seal()
	return err
}

// Local reports whether this endpoint can only be reached from this machine.
func Local(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// behind refuses anything that does not present the token.
//
// On the loopback interface this is the second line, behind a port nothing
// outside the machine can reach. Anywhere else it is the only one.
func behind(token string, next http.Handler) http.Handler {
	want := []byte("Bearer " + token)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := r.Header.Get("Authorization")
		if subtle.ConstantTimeCompare([]byte(got), want) != 1 {
			w.Header().Set("WWW-Authenticate", `Bearer realm="numen"`)
			http.Error(w, "this vault is not open to you", http.StatusUnauthorized)
			return
		}
		if !strings.HasPrefix(r.URL.Path, "/mcp") {
			http.NotFound(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}
