//go:build !nomcp

package agents

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/claudecode"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/agent"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/mcp"
	"github.com/jiva-studio/numen/modules/libs/core/container"
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
	// Reads serves the surface every tool of which reads. An agent answering
	// from it changes nothing.
	Reads bool
	// Addr is where the tools are served. Empty takes a loopback port this
	// machine picks.
	Addr string
	// Token is what an agent presents. Empty mints one and keeps it beside the
	// vault list.
	Token string
	// Announcing writes down where the tools are and what to present, which is
	// what an agent a person runs themselves is configured from. The file names
	// one vault, and one window writes it.
	Announcing bool
	// Root is the folder the agent is started in.
	Root string
	// Drafting is how a change the agent is making is drawn before it lands.
	Drafting claudecode.Drafting
	// Out is where what happened is said.
	Out io.Writer
}

// Served is the tools on a port, and the agent the settings name reaching them.
type Served struct {
	// Agent is the agent this window asks on the person's behalf, and nothing
	// where the settings name none.
	Agent *claudecode.Agent
	// URL is where an agent reaches the tools, naming the port that was bound.
	URL string

	shut func() error
}

// Close takes the agents away and then the endpoint.
func (s *Served) Close() error {
	if s == nil || s.shut == nil {
		return nil
	}
	return s.shut()
}

// Serve puts the tools on a port and starts the agent the settings name
// against them.
func Serve(ctx context.Context, opts Options) (*Served, error) {
	secret := opts.Token
	if secret == "" {
		minted, err := Token(opts.Config)
		if err != nil {
			return nil, err
		}
		secret = minted
	}
	addr := opts.Addr
	if addr == "" {
		addr = ephemeral
	}

	trouble := func(err error) { fmt.Fprintln(opts.Out, "agents:", err) }
	serving := mcp.ServeHTTP
	if opts.Reads {
		serving = mcp.ServeReadingHTTP
	}
	endpoint, err := serving(ctx, addr, secret, opts.Core, trouble)
	if err != nil {
		return nil, err
	}
	forget := func() {}
	if opts.Announcing {
		gone, err := Announce(opts.Config, endpoint.URL, secret)
		if err != nil {
			endpoint.Close(context.Background())
			return nil, err
		}
		forget = gone
	}

	fmt.Fprintf(opts.Out, "agents: %s\n", endpoint.URL)
	if !mcp.Local(addr) {
		fmt.Fprintf(opts.Out, "agents: %s is reachable from the network, not only from this machine\n", addr)
	}

	served := &Served{URL: endpoint.URL}
	if opts.Config.Agent.Use == agent.UseClaude {
		// What the window says about a call is what the tool declared about
		// itself, asked for over the protocol an agent is answered by.
		vocabulary := mcp.Vocabulary
		if opts.Reads {
			vocabulary = mcp.ReadingVocabulary
		}
		words, err := vocabulary(ctx, opts.Core)
		if err != nil {
			fmt.Fprintln(opts.Out, "agents:", err)
		}
		served.Agent = Claude(opts.Config, opts.Root, endpoint.URL, secret, words, opts.Drafting, opts.Out)
	}

	started := served.Agent
	served.shut = func() error {
		forget()
		// The agents this window started go first: each is in a process group
		// of its own, so nothing else reaches them, and one still answering
		// would go on writing to the vault after the window is gone.
		var stopped error
		if started != nil {
			stopped = started.Close()
		}
		shutdown, cancel := context.WithTimeout(context.Background(), bound)
		defer cancel()
		if err := endpoint.Close(shutdown); err != nil {
			return err
		}
		return stopped
	}
	return served, nil
}

// Claude is what the window asks on the person's behalf.
//
// It reaches the same tools over the same port as an agent somebody configured
// themselves, and is given all of them: what it changes appears in the window
// as it happens.
func Claude(
	cfg container.Config,
	root, url, secret string,
	served map[string]mcp.Words,
	drafting claudecode.Drafting,
	out io.Writer,
) *claudecode.Agent {
	words := make(map[string]claudecode.Words, len(served))
	for name, said := range served {
		words[claudecode.Tool(name)] = claudecode.Words{
			Title:   said.Title,
			About:   said.About,
			Inside:  said.Inside,
			Kind:    said.Kind,
			Stood:   said.Stood,
			Becomes: said.Becomes,
		}
	}
	return &claudecode.Agent{
		Command:             cfg.Agent.Claude.Command,
		Root:                root,
		Tools:               claudecode.Endpoint{URL: url, Token: secret},
		Allowed:             []string{claudecode.Tool("*")},
		Words:               words,
		Drafting:            drafting,
		Model:               cfg.Agent.Claude.Model,
		Turns:               cfg.Agent.Claude.MaxSteps,
		ReadsHooksAndSkills: cfg.Agent.Claude.ReadsHooksAndSkills,
		Trouble:             func(err error) { fmt.Fprintln(out, "agent:", err) },
	}
}
