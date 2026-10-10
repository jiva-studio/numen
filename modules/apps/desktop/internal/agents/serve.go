//go:build !nomcp

package agents

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"sync"
	"time"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/acp"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/claudecode"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/agent"
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
	// Drafting is how a change the agent is making is drawn before it lands.
	Drafting claudecode.Drafting
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
	secret := opts.Token
	if secret == "" {
		token, err := GetToken(opts.Config)
		if err != nil {
			return nil, err
		}
		secret = token
	}
	addr := opts.Addr
	if addr == "" {
		addr = ephemeral
	}

	errorHandler := func(err error) { fmt.Fprintln(opts.Out, "agents:", err) }
	serving := mcp.ServeHTTP
	switch {
	case opts.ShouldRead:
		serving = mcp.ServeReadingHTTP
	case opts.ShouldReview:
		serving = mcp.ServeReviewingHTTP
	}
	endpoint, err := serving(ctx, addr, secret, opts.Core, errorHandler)
	if err != nil {
		return nil, err
	}
	forget := func() {}
	if opts.IsAnnouncing {
		gone, err := Announce(opts.Config, opts.Root, endpoint.URL, secret)
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

	vocabulary := mcp.Vocabulary
	switch {
	case opts.ShouldRead:
		vocabulary = mcp.ReadingVocabulary
	case opts.ShouldReview:
		vocabulary = mcp.GetReviewVocabulary
	}
	words, err := vocabulary(ctx, opts.Core)
	if err != nil {
		fmt.Fprintln(opts.Out, "agents:", err)
	}

	router := &Router{
		opts:       opts,
		url:        endpoint.URL,
		secret:     secret,
		vocabulary: words,
	}

	served := &Server{
		URL: endpoint.URL,
	}
	if opts.Config.Agent.Use != "" {
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

// Router routes tasks to the configured agent dynamically.
type Router struct {
	opts       Options
	url        string
	secret     string
	vocabulary map[string]mcp.Tool

	mu     sync.Mutex
	closed bool
	claude *claudecode.Agent
	anti   *acp.Agent
	codex  *acp.Agent
}

// Close stops every agent router managed.
func (r *Router) Close() error {
	r.mu.Lock()
	r.closed = true
	var errs []error
	if r.claude != nil {
		if err := r.claude.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if r.anti != nil {
		if err := r.anti.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if r.codex != nil {
		if err := r.codex.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	r.mu.Unlock()
	return errors.Join(errs...)
}

func (r *Router) getAgent(use string, cfg container.Config) port.Agent {
	r.mu.Lock()
	defer r.mu.Unlock()

	switch use {
	case agent.UseAntigravity:
		if r.anti == nil {
			r.anti = &acp.Agent{
				Program:      "antigravity",
				Command:      cfg.Agent.Antigravity.Command,
				Root:         r.opts.Root,
				Tools:        acp.Endpoint{URL: r.url, Token: r.secret},
				Model:        cfg.Agent.Antigravity.Model,
				Turns:        cfg.Agent.Antigravity.MaxSteps,
				ErrorHandler: func(err error) { fmt.Fprintln(r.opts.Out, "agent:", err) },
			}
		} else {
			r.anti.Model = cfg.Agent.Antigravity.Model
			r.anti.Turns = cfg.Agent.Antigravity.MaxSteps
			if len(cfg.Agent.Antigravity.Command) > 0 {
				r.anti.Command = cfg.Agent.Antigravity.Command
			}
		}
		return r.anti
	case agent.UseCodex:
		if r.codex == nil {
			r.codex = &acp.Agent{
				Program:      "codex",
				Command:      cfg.Agent.Codex.Command,
				Root:         r.opts.Root,
				Tools:        acp.Endpoint{URL: r.url, Token: r.secret},
				Model:        cfg.Agent.Codex.Model,
				Turns:        cfg.Agent.Codex.MaxSteps,
				ErrorHandler: func(err error) { fmt.Fprintln(r.opts.Out, "agent:", err) },
			}
		} else {
			r.codex.Model = cfg.Agent.Codex.Model
			r.codex.Turns = cfg.Agent.Codex.MaxSteps
			if len(cfg.Agent.Codex.Command) > 0 {
				r.codex.Command = cfg.Agent.Codex.Command
			}
		}
		return r.codex
	default:
		if r.claude == nil {
			r.claude = Claude(cfg, r.opts.Root, r.url, r.secret, r.vocabulary, r.opts.Drafting, r.opts.Out)
		} else {
			r.claude.Model = cfg.Agent.Claude.Model
			r.claude.Turns = cfg.Agent.Claude.MaxSteps
		}
		return r.claude
	}
}

func (r *Router) currentConfig() (string, container.Config) {
	use := r.opts.Config.Agent.Use
	if use == "" {
		use = agent.UseClaude
	}
	return use, r.opts.Config
}

// Take starts the task on the active agent.
func (r *Router) Take(ctx context.Context, task port.Task) (port.Run, error) {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil, errors.New("agent is closed")
	}
	r.mu.Unlock()

	use, cfg := r.currentConfig()
	if task.Agent != "" {
		use = task.Agent
	}
	fmt.Fprintf(r.opts.Out, "agents: dispatching task to %s (model: %s)\n", use, task.Model)
	agentRunner := r.getAgent(use, cfg)
	if agentRunner == nil {
		return nil, errors.New("no agent configured")
	}
	return agentRunner.Take(ctx, task)
}

// Finish forwards conversation end to all active agents.
func (r *Router) Finish(ctx context.Context, conversation string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var errs []error
	if r.claude != nil {
		if err := r.claude.Finish(ctx, conversation); err != nil {
			errs = append(errs, err)
		}
	}
	if r.anti != nil {
		if err := r.anti.Finish(ctx, conversation); err != nil {
			errs = append(errs, err)
		}
	}
	if r.codex != nil {
		if err := r.codex.Finish(ctx, conversation); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// Claude is what the window asks on the person's behalf.
func Claude(
	cfg container.Config,
	root, url, secret string,
	vocabulary map[string]mcp.Tool,
	drafting claudecode.Drafting,
	out io.Writer,
) *claudecode.Agent {
	words := make(map[string]claudecode.ToolDeclaration, len(vocabulary))
	allowed := make([]string, 0, len(vocabulary))
	for name, said := range vocabulary {
		allowed = append(allowed, claudecode.Tool(name))
		words[claudecode.Tool(name)] = claudecode.ToolDeclaration{
			Title: said.Title,
			Kind:  said.Kind,
			Arguments: claudecode.Arguments{
				About:   said.About,
				Element: said.Inside,
				Match:   said.Match,
				Text:    said.Text,
			},
		}
	}
	slices.Sort(allowed)
	return &claudecode.Agent{
		Command:                  cfg.Agent.Claude.Command,
		Root:                     root,
		Tools:                    claudecode.Endpoint{URL: url, Token: secret},
		Allowed:                  allowed,
		Words:                    words,
		Drafting:                 drafting,
		Model:                    cfg.Agent.Claude.Model,
		Turns:                    cfg.Agent.Claude.MaxSteps,
		ShouldReadHooksAndSkills: cfg.Agent.Claude.ShouldReadHooksAndSkills,
		ErrorHandler:             func(err error) { fmt.Fprintln(out, "agent:", err) },
	}
}
