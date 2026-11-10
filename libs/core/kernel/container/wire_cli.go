package container

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/cli"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/port/agentport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// OpenCLIVaults builds the use cases of the vault commands over this installation's list. The
// commands that change what the index remembers open it when they run.
func (c Config) OpenCLIVaults() (cli.Vaults, error) {
	ports, err := c.Registry()
	if err != nil {
		return cli.Vaults{}, err
	}
	unregister := func(db *Index) vault.Unregister {
		return vault.NewUnregister(ports.Vaults, ports.Remover, db.Vaults())
	}
	return cli.Vaults{
		Find:       registry.NewFind(ports.Lookup),
		Add:        vault.NewAdd(c.VaultIdentity(), ports.Vaults, ports.Lookup, ports.Saver, c.Clock()),
		Known:      registry.NewGetKnownVaults(ports.Vaults, ports.Last, c.VaultReaders()),
		RecordOpen: vault.NewRecordOpen(ports.Recorder),
		Now:        c.Clock(),
		OpenRename: func(ctx context.Context) (cli.Renaming, error) {
			db, err := c.OpenIndex(ctx)
			if err != nil {
				return cli.Renaming{}, err
			}
			return cli.Renaming{Rename: vault.NewRename(ports.Vaults, ports.Saver, db.Vaults()), Close: db.Close}, nil
		},
		OpenUnregister: func(ctx context.Context) (cli.Unregistering, error) {
			db, err := c.OpenIndex(ctx)
			if err != nil {
				return cli.Unregistering{}, err
			}
			return cli.Unregistering{Unregister: unregister(db), Close: db.Close}, nil
		},
		OpenErase: func(ctx context.Context) (cli.Erasing, error) {
			db, err := c.OpenIndex(ctx)
			if err != nil {
				return cli.Erasing{}, err
			}
			erase := vault.NewErase(c.VaultIdentity(), c.Trash(), unregister(db))
			return cli.Erasing{Erase: erase, Close: db.Close}, nil
		},
	}, nil
}

// OpenCLILinks builds the link query over the index. The caller closes it.
func (c Config) OpenCLILinks(ctx context.Context) (cli.Links, error) {
	db, err := c.OpenIndex(ctx)
	if err != nil {
		return cli.Links{}, err
	}
	notes := c.Notes(db, c.UpdateIndex(db))
	return cli.Links{Show: notes.Links, Close: db.Close}, nil
}

// OpenCLIScan builds the reading of one vault into the index, with the vectors when a model is
// installed. A rebuild reads every file again.
func (c Config) OpenCLIScan(ctx context.Context, v domain.Vault, rebuild bool) (cli.Scan, error) {
	indexing := c
	indexing.ShouldRebuildIndex = rebuild
	db, err := indexing.OpenIndex(ctx)
	if err != nil {
		return cli.Scan{}, err
	}
	indexing, err = indexing.OpenPluginWork(ctx, db, v)
	if err != nil {
		return cli.Scan{}, errors.Join(err, db.Close())
	}
	embedder, closeEmbedder, why := indexing.Embedder(ctx)
	read, err := indexing.ReadWholeVault(ctx, db, embedder, v)
	if err != nil {
		return cli.Scan{}, errors.Join(err, closeIfOpen(closeEmbedder), db.Close())
	}
	return cli.Scan{
		Read: read,
		Summary: func(ctx context.Context) (domain.VaultSummary, error) {
			return db.Summaries().GetSummary(ctx, v.ID)
		},
		Unembedded: why,
		Close: func() error {
			return errors.Join(closeIfOpen(closeEmbedder), db.Close())
		},
	}, nil
}

// OpenCLISearch builds the search the window runs. Without a model it searches by words, and
// says why in the Words field.
func (c Config) OpenCLISearch(
	ctx context.Context, v domain.Vault, errorHandler window.ErrorHandler,
) (cli.Search, error) {
	db, err := c.OpenIndex(ctx)
	if err != nil {
		return cli.Search{}, err
	}
	asking, err := c.OpenPluginWork(ctx, db, v)
	if err != nil {
		return cli.Search{}, errors.Join(err, db.Close())
	}
	embedder, closeEmbedder, why := asking.Embedder(ctx)
	return cli.Search{
		Search: asking.NewSearch(db, embedder, errorHandler),
		Words:  why,
		Close: func() error {
			return errors.Join(closeIfOpen(closeEmbedder), db.Close())
		},
	}, nil
}

// OpenCLIProblems builds the standard checks over the index. The caller closes it.
func (c Config) OpenCLIProblems(ctx context.Context) (cli.Problems, error) {
	db, err := c.OpenIndex(ctx)
	if err != nil {
		return cli.Problems{}, err
	}
	return cli.Problems{Problems: c.GetProblems(db.Problems()), Close: db.Close}, nil
}

// GetCLIPluginCommands builds the commands the enabled plugins declare, and what readies the
// plugins to run one.
func (c Config) GetCLIPluginCommands() cli.PluginCommands {
	return cli.PluginCommands{List: c.GetPluginCommands().List, Open: c.OpenPluginCommands}
}

func closeIfOpen(closeFn func() error) error {
	if closeFn == nil {
		return nil
	}
	return closeFn()
}

// GetVaultList builds the use case that lists the vaults of the registry given.
func (c Config) GetVaultList(ports registry.Ports) registry.List {
	return registry.NewList(ports.Vaults)
}

// GetAgentFunc returns the agent of the installation to a window that asks for it on each call.
func (c Config) GetAgentFunc() func() agentport.Agent {
	agent := c.GetSettingsAgent()
	return func() agentport.Agent { return agent }
}
