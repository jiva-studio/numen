// Package service is the backend of the tags plugin: it builds the host clients and the cache of the
// tags, and registers the command that updates the tags of notes and the tool tag_list.
package service

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/command"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"

	"github.com/jiva-studio/numen/plugins/tools-taxonomy/backend/tagcache"
)

// UpdateCommand is the command that brings the tags of the vault up to date with its file change feed.
// A trigger on note.frontmatterChanged runs it; the notes the trigger names are read from the feed.
const UpdateCommand = "tags.update"

// ShownVault names the vault the host shows, which a command runs over.
type ShownVault interface {
	GetVault(ctx context.Context) (domain.Vault, error)
}

// Host is what the plugin reaches of the host: the vault it shows and the frontmatter and the store
// folder of each vault.
type Host struct {
	Shown ShownVault
	Open  func(id domain.VaultID) tagcache.Vault
}

func openHostAt(endpoint hostcall.Endpoint) Host {
	client := host.NewClient(endpoint)
	store := host.NewStoreClient(endpoint)
	return Host{
		Shown: client,
		Open: func(id domain.VaultID) tagcache.Vault {
			vault := client.With(id)
			return tagcache.Vault{Frontmatter: vault, Feed: vault, Store: store.With(id)}
		},
	}
}

// Mount registers the command and the tool on s, over the host the environment names.
func Mount(s *server.Server) error {
	endpoint, err := hostcall.FromEnv()
	if err != nil {
		return err
	}
	commands, registry, err := NewRegistries(openHostAt(endpoint))
	if err != nil {
		return err
	}
	command.Mount(s, commands)
	registry.Mount(s)
	return nil
}

// NewRegistries returns the command and the tool of the plugin over h, sharing one cache of tags.
func NewRegistries(h Host) (*command.Registry, *tools.Registry, error) {
	cache := tagcache.New(h.Open)
	commands := command.New()
	if err := commands.Add(UpdateCommand, newUpdateCommand(h.Shown, cache)); err != nil {
		return nil, nil, err
	}
	registry, err := newTools(cache)
	if err != nil {
		return nil, nil, err
	}
	return commands, registry, nil
}

func newUpdateCommand(shown ShownVault, cache *tagcache.Caches) command.Run {
	return func(ctx context.Context, _ command.Request) (command.Result, error) {
		v, err := shown.GetVault(ctx)
		if err != nil {
			return command.Result{}, err
		}
		if err := cache.UpdateVault(ctx, v.ID); err != nil {
			return command.Result{}, err
		}
		return command.Result{}, nil
	}
}
