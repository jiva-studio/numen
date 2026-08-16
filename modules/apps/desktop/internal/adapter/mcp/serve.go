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
	// stop lets go of the goroutine waiting on the context, so that closing one
	// endpoint does not leave a watcher behind until the application exits.
	stop func()
	once sync.Once
}

// ServeHTTP starts the server and returns once it is listening.
//
// The listener is opened before returning, so a port already in use is an
// error the person sees at startup.
//
// Trouble, if it is given, is called with whatever stops the server later. It
// is how the person hears that it stopped.
func ServeHTTP(ctx context.Context, addr, token string, core Core, trouble func(error)) (*Endpoint, error) {
	if addr == "" {
		addr = DefaultAddr
	}
	if token == "" {
		return nil, errors.New("a server an agent can reach needs a token to present")
	}

	server := New(core)
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

	endpoint := &Endpoint{
		URL: "http://" + listener.Addr().String() + "/mcp",
		server: &http.Server{
			Handler:           behind(token, handler),
			ReadHeaderTimeout: 10 * time.Second,
		},
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
			endpoint.Close()
		case <-stopped:
		}
	}()
	return endpoint, nil
}

// Close stops answering. An agent in the middle of a call is cut off, which is
// what closing the application means.
func (e *Endpoint) Close() error {
	if e == nil || e.server == nil {
		return nil
	}
	if e.stop != nil {
		e.once.Do(e.stop)
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return e.server.Shutdown(shutdown)
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

// ServeStdio answers on this process's own pipes, for an agent that starts the
// server itself. It returns when the agent disconnects.
func ServeStdio(ctx context.Context, core Core) error {
	return New(core).Run(ctx, &sdk.StdioTransport{})
}
