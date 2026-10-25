package derivedfile

import (
	"context"
	"fmt"
	"strings"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// PluginStores opens a plugin's folder in the store of a vault, `.numen/<plugin id>/`.
type PluginStores struct {
	Options vaultconfig.Options
}

// Open opens the folder of the plugin. A plugin's identity always holds a dot, so its folder is
// never one of the store's own areas.
func (p PluginStores) Open(v domain.Vault, pluginID string) (pluginport.PluginStore, error) {
	if !strings.Contains(pluginID, ".") || strings.HasPrefix(pluginID, ".") || strings.ContainsAny(pluginID, `/\`) {
		return nil, fmt.Errorf("%q is not the identity of a plugin: %w", pluginID, vaultconfig.ErrOutside)
	}
	store, err := OpenDerived(v.Path, p.Options, pluginID)
	if err != nil {
		return nil, err
	}
	return pluginFolder{store: store, folder: pluginID}, nil
}

// pluginFolder is one plugin's folder of a vault's store, whose files are named without it.
type pluginFolder struct {
	store  *DerivedStore
	folder string
}

func (f pluginFolder) getKey(name string) (string, error) {
	if name == "" || strings.ContainsAny(name, `/\`) || strings.HasPrefix(name, ".") {
		return "", fmt.Errorf("%q is not a file of the folder: %w", name, vaultconfig.ErrOutside)
	}
	return f.folder + "/" + name, nil
}

func (f pluginFolder) Read(ctx context.Context, name string) ([]byte, error) {
	key, err := f.getKey(name)
	if err != nil {
		return nil, err
	}
	return f.store.Read(ctx, key)
}

func (f pluginFolder) Append(ctx context.Context, name string, content []byte) error {
	key, err := f.getKey(name)
	if err != nil {
		return err
	}
	return f.store.Append(ctx, key, content)
}

func (f pluginFolder) List(ctx context.Context) ([]pluginport.PluginStoreFile, error) {
	found, err := f.store.List(ctx, f.folder)
	if err != nil {
		return nil, err
	}
	out := make([]pluginport.PluginStoreFile, 0, len(found))
	for _, one := range found {
		out = append(out, pluginport.PluginStoreFile{Name: strings.TrimPrefix(one.Name, f.folder+"/"), Size: int64(one.Size)})
	}
	return out, nil
}

func (f pluginFolder) Write(ctx context.Context, name string, content []byte) error {
	key, err := f.getKey(name)
	if err != nil {
		return err
	}
	return f.store.Write(ctx, key, content)
}

func (f pluginFolder) Delete(ctx context.Context, name string) error {
	key, err := f.getKey(name)
	if err != nil {
		return err
	}
	return f.store.Remove(ctx, key)
}
