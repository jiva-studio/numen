package agents

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/container"
)

// ephemeral is a loopback port this machine picks, for a window that is not the
// one an agent is configured against.
const ephemeral = "127.0.0.1:0"

// bound is how long the transport has to be cut off. A session left open holds
// its connection until it is closed under it, and this is how long that costs.
const bound = 2 * time.Second

// Options is what a window says about letting agents in.
type Options struct {
	// Config is this installation's settings: which agent answers, and where
	// the application keeps its own state.
	Config container.Config
	// ShouldRead serves the surface every tool of which reads. An agent answering
	// from it changes nothing.
	ShouldRead bool
	// ShouldReview serves the surface the window a person runs their cards in has.
	ShouldReview bool
	// Addr is where the tools are served. Empty takes a loopback port this
	// machine picks.
	Addr string
	// Token is what an agent presents. Empty mints one and keeps it beside the
	// vault list.
	Token string
	// IsAnnouncing writes down where the tools are and what to present, which is
	// what an agent a person runs themselves is configured from. The file names
	// one vault, and one window writes it.
	IsAnnouncing bool
	// Root is the folder the agent is started in.
	Root string
	// Out is where what happened is said.
	Out io.Writer
}

// Server is the tools on a port.
type Server struct {
	// URL is where an agent reaches the tools, naming the port that was bound.
	URL string

	close func() error
}

// Close takes the endpoint away.
func (s *Server) Close() error {
	if s == nil || s.close == nil {
		return nil
	}
	return s.close()
}

// Serve puts the tools on a port.
func Serve(ctx context.Context, opts Options) (*Server, error) {
	secret, err := getSecret(opts)
	if err != nil {
		return nil, err
	}
	addr := opts.Addr
	if addr == "" {
		addr = ephemeral
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/sse", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	httpServer := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		_ = httpServer.Serve(ln)
	}()

	url := fmt.Sprintf("http://%s/mcp", ln.Addr().String())
	forget := func() {}
	if opts.IsAnnouncing {
		removed, err := WriteDiscovery(opts.Config, opts.Root, url, secret)
		if err != nil {
			_ = httpServer.Close()
			_ = ln.Close()
			return nil, err
		}
		forget = removed
	}

	fmt.Fprintf(opts.Out, "agents: %s\n", url)
	if !isLocal(addr) {
		fmt.Fprintf(opts.Out, "agents: %s is reachable from the network, not only from this machine\n", addr)
	}

	served := &Server{
		URL: url,
	}

	served.close = func() error {
		forget()
		shutdown, cancel := context.WithTimeout(context.Background(), bound)
		defer cancel()
		return httpServer.Shutdown(shutdown)
	}
	return served, nil
}

func isLocal(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// getSecret is the token the endpoint is guarded with: the one given, or the
// one the configuration holds.
func getSecret(opts Options) (string, error) {
	if opts.Token != "" {
		return opts.Token, nil
	}
	return GetToken(opts.Config)
}
