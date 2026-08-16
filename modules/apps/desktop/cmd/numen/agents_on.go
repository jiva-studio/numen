//go:build !nomcp

package main

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/mcp"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/webui"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/container"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/lint"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/note"
)

// This file is the only one that knows an agent can reach the vault. Built with
// `nomcp`, its counterpart takes its place and nothing of the tools — nor of
// what serves them — is linked in at all.

const defaultAgentAddr = mcp.DefaultAddr

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
	endpoint, err := mcp.ServeHTTP(ctx, opts.addr, secret, agentCore(cfg, opened, root),
		func(err error) { fmt.Fprintln(out, "agents:", err) })
	if err != nil {
		return nil, err
	}
	forget, err := announce(cfg, endpoint.URL, secret)
	if err != nil {
		endpoint.Close()
		return nil, err
	}

	fmt.Fprintf(out, "agents: %s\n", endpoint.URL)
	if !mcp.Local(opts.addr) {
		fmt.Fprintf(out, "agents: %s is reachable from the network, not only from this machine\n", opts.addr)
	}
	return func() error {
		forget()
		return endpoint.Close()
	}, nil
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

	return mcp.Core{
		Vault:   opened.Vault,
		Root:    root,
		Readers: readers,
		Notes:   queries,

		Search:        note.Search{Notes: queries},
		Neighbourhood: note.ShowNeighbourhood{Links: opened.Index.Links(), Notes: queries},
		Links:         note.ShowLinks{Links: opened.Index.Links()},
		Problems:      lint.Standard(opened.Index.Problems()),

		Create: note.Create{
			Writers: writers, Names: queries, Index: index,
			Extension: extension(cfg),
		},
		Write:   note.Write{Readers: readers, Writers: writers, Index: index},
		Move:    note.Move{Readers: readers, Writers: writers, Links: opened.Index.Links(), Index: index},
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
