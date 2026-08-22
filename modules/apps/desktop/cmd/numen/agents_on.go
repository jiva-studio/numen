//go:build !nomcp

package main

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/agent/claudecode"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/mcp"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/webui"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/container"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/lint"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/markdown"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/note"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/search"
)

// This file is the only one that knows an agent can reach the vault. Built with
// `nomcp`, its counterpart takes its place and nothing of the tools — nor of
// what serves them — is linked in at all.

const defaultAgentAddr = mcp.DefaultAddr

// agentBound is how long the agents' transport has to be cut off. A session an
// agent left open holds its connection until it is closed under it, and this is
// how long that costs. The calls already running are waited for afterwards,
// without a bound.
const agentBound = 2 * time.Second

func serveAgents(ctx context.Context, cfg container.Config, opened *webui.Opened, opts agentOptions, out io.Writer) (func() error, error) {
	if opts.off {
		return func() error { return nil }, nil
	}

	root, err := filepath.Abs(opened.Vault.Path)
	if err != nil {
		return nil, err
	}

	secret, err := token(cfg)
	if err != nil {
		return nil, err
	}
	core := agentCore(cfg, opened, root)
	endpoint, err := mcp.ServeHTTP(ctx, opts.addr, secret, core,
		func(err error) { fmt.Fprintln(out, "agents:", err) })
	if err != nil {
		return nil, err
	}
	forget, err := announce(cfg, endpoint.URL, secret)
	if err != nil {
		endpoint.Close(context.Background())
		return nil, err
	}

	fmt.Fprintf(out, "agents: %s\n", endpoint.URL)
	if !mcp.Local(opts.addr) {
		fmt.Fprintf(out, "agents: %s is reachable from the network, not only from this machine\n", opts.addr)
	}

	// What the window says about a call is what the tool declared about
	// itself, asked for over the protocol an agent is answered by.
	words, err := mcp.Vocabulary(ctx, core)
	if err != nil {
		fmt.Fprintln(out, "agents:", err)
	}
	// What the agent is about to change reaches the window before the change
	// does, and where a stretch stands is the vault's to say.
	reading := note.Read{Readers: cfg.VaultReaders()}
	drafting := claudecode.Drafting{
		Tell: func(ctx context.Context, said domain.Editing) {
			_ = opened.API.Viewing().Editing(ctx, said)
		},
		Where: func(ctx context.Context, path, stood string) (int, int, bool) {
			contents, err := reading.Execute(ctx, opened.Vault, path)
			if err != nil || contents.Outcome != note.Ok {
				return 0, 0, false
			}
			at, _ := markdown.Where(contents.Body, stood)
			if len(at) != 1 {
				return 0, 0, false
			}
			return markdown.Counted(contents.Body, at[0].From),
				markdown.Counted(contents.Body, at[0].To), true
		},
	}
	started := agent(cfg, root, endpoint.URL, secret, words, drafting, out)
	opened.API.Agent = started

	return func() error {
		forget()
		// The agents this window started go first: each is in a process group
		// of its own, so nothing else reaches them, and one still answering
		// would go on writing to the vault after the window is gone.
		stopped := started.Close()
		shutdown, cancel := context.WithTimeout(context.Background(), agentBound)
		defer cancel()
		if err := endpoint.Close(shutdown); err != nil {
			return err
		}
		return stopped
	}, nil
}

// agent is what the window asks on the person's behalf.
//
// It reaches the same tools over the same port as an agent somebody configured
// themselves, and is given all of them: what it changes appears in the window
// as it happens.
func agent(
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

// agentCore wires the tools to the same use cases everything else uses. The
// index is brought level by the same refresh the watcher drives, so a tool that
// writes a note leaves it findable.
func agentCore(cfg container.Config, opened *webui.Opened, root string) mcp.Core {
	index := func(ctx context.Context, v domain.Vault, paths []string) error {
		_, err := opened.Refresh.Execute(ctx, v, paths)
		return err
	}
	readers := cfg.VaultReaders()
	writers := cfg.VaultWriters()
	queries := opened.Index.Queries()
	viewing := opened.API.Viewing()
	// What a write is doing reaches the window the way a note put in front of
	// the person does. A build with no window draws nothing and is told nothing.
	tells := note.Telling(func(ctx context.Context, said domain.Editing) {
		if viewing == nil {
			return
		}
		_ = viewing.Editing(ctx, said)
	})

	return mcp.Core{
		Vault:   opened.Vault,
		Root:    root,
		Readers: readers,
		View:    opened.API.Viewing(),
		Notes:   queries,

		Sources:   opened.Index.SourcesKnown(),
		Recognise: recogniser(opened),
		Derived:   cfg.DerivedStores(),

		Search:        search.New(opened.Index.Passages(), readers, cfg.DerivedStores(), opened.Embedder, cfg.Embedding.Floor),
		Neighbourhood: note.ShowNeighbourhood{Links: opened.Index.Links(), Notes: queries},
		Links:         note.ShowLinks{Links: opened.Index.Links()},
		Problems:      lint.Standard(opened.Index.Problems()),

		Create: note.Create{
			Writers: writers, Names: queries, Index: index,
			Extension: extension(cfg),
		},
		Write:   note.Write{Readers: readers, Writers: writers, Index: index, Telling: tells},
		Replace: note.Replace{Readers: readers, Writers: writers, Index: index, Telling: tells},
		Move: note.Move{
			Readers: readers, Writers: writers, Links: opened.Index.Links(), Index: index,
			Moving: func(ctx context.Context, went domain.Went) {
				_ = opened.API.Viewing().Moved(ctx, went)
			},
		},
		Remove:  note.Remove{Readers: readers, Writers: writers, Links: opened.Index.Links(), Index: index},
		Linking: note.Linking{Readers: readers, Writers: writers, Index: index},
	}
}

// extension is what a note this vault holds is filed under.
func extension(cfg container.Config) string {
	if len(cfg.Extensions) > 0 {
		return cfg.Extensions[0]
	}
	return ""
}

// recogniser is what reads a scanned document for an agent that asks.
//
// It is always served, even on a machine holding none of the models: what is
// missing is fetched behind whoever asked, and the tool says so. A tool that is
// not served at all leaves an agent saying the vault cannot do a thing it can.
func recogniser(opened *webui.Opened) mcp.Recognising {
	return opened.Recognising
}
