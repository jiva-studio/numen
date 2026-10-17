//go:build !nomcp

package agents

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/mcp"
	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// ephemeral is a loopback port this machine picks, for a window that is not the
// one an agent is configured against.
const ephemeral = "127.0.0.1:0"

// bound is how long the agents' transport has to be cut off. A session an agent
// left open holds its connection until it is closed under it, and this is how
// long that costs. The calls already running are waited for afterwards, without
// a bound.
const bound = 2 * time.Second

// Options is what a window says about letting agents in.
type Options struct {
	// Config is this installation's settings: which agent answers, and where
	// the application keeps its own state.
	Config container.Config
	// Core is everything the tools work through.
	Core mcp.Core
	// ShouldRead serves the surface every tool of which reads. An agent answering
	// from it changes nothing.
	ShouldRead bool
	// ShouldReview serves the surface the window a person runs their cards in has:
	// everything that reads, and the cards of a deck.
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
	// Preview is how a change the agent is making is drawn before it lands.
	Preview EditPreview
	// Provider runs the tasks, in place of the one the plugins make.
	Provider port.AgentProvider
	// LimitedToolsOnly refuses the programs that run tools of their own.
	LimitedToolsOnly bool
	// Out is where what happened is said.
	Out io.Writer
}

// Server is the tools on a port, and the agent the settings name reaching them.
type Server struct {
	// Agent is the agent this window asks on the person's behalf, and nothing
	// where the settings name none.
	Agent port.Agent
	// URL is where an agent reaches the tools, naming the port that was bound.
	URL string

	close func() error
}

// Close takes the agents away and then the endpoint.
func (s *Server) Close() error {
	if s == nil || s.close == nil {
		return nil
	}
	return s.close()
}

// Serve puts the tools on a port and starts the agent the settings name
// against them.
func Serve(ctx context.Context, opts Options) (*Server, error) {
	secret, err := getSecret(opts)
	if err != nil {
		return nil, err
	}
	addr := opts.Addr
	if addr == "" {
		addr = ephemeral
	}

	errorHandler := func(err error) { fmt.Fprintln(opts.Out, "agents:", err) }
	endpoint, err := getServeFunc(opts)(ctx, addr, secret, opts.Core, errorHandler)
	if err != nil {
		return nil, err
	}
	forget := func() {}
	if opts.IsAnnouncing {
		gone, err := WriteDiscovery(opts.Config, opts.Root, endpoint.URL, secret)
		if err != nil {
			//nolint:contextcheck // an endpoint taken down again is closed whatever became of the context it opened under
			endpoint.Close(context.Background())
			return nil, err
		}
		forget = gone
	}

	fmt.Fprintf(opts.Out, "agents: %s\n", endpoint.URL)
	if !mcp.Local(addr) {
		fmt.Fprintf(opts.Out, "agents: %s is reachable from the network, not only from this machine\n", addr)
	}

	words, err := getVocabularyFunc(opts)(ctx, opts.Core)
	if err != nil {
		fmt.Fprintln(opts.Out, "agents:", err)
	}

	router := NewRouter(opts, endpoint.URL, secret, words)

	served := &Server{
		URL: endpoint.URL,
	}
	if opts.Config.Agent.IsNamed() {
		served.Agent = router
	}

	//nolint:contextcheck // a close runs when the context is already over, so it carries one of its own with a bound
	served.close = func() error {
		forget()
		stopped := router.Close()
		shutdown, cancel := context.WithTimeout(context.Background(), bound)
		defer cancel()
		if err := endpoint.Close(shutdown); err != nil {
			return err
		}
		return stopped
	}
	return served, nil
}

// getSecret is the token the endpoint is guarded with: the one given, or the
// one the configuration holds.
func getSecret(opts Options) (string, error) {
	if opts.Token != "" {
		return opts.Token, nil
	}
	return GetToken(opts.Config)
}

// getServeFunc is the function that serves the tools the options select.
func getServeFunc(
	opts Options,
) func(context.Context, string, string, mcp.Core, func(error)) (*mcp.Endpoint, error) {
	switch {
	case opts.ShouldRead:
		return mcp.ServeReadingHTTP
	case opts.ShouldReview:
		return mcp.ServeReviewHTTP
	}
	return mcp.ServeHTTP
}

// getVocabularyFunc is the function that lists the tools the options select.
func getVocabularyFunc(opts Options) func(context.Context, mcp.Core) (map[string]mcp.Tool, error) {
	switch {
	case opts.ShouldRead:
		return mcp.ReadingVocabulary
	case opts.ShouldReview:
		return mcp.GetReviewVocabulary
	}
	return mcp.Vocabulary
}
