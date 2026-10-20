package filesystem

import (
	"context"
	"fmt"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// PluginData opens the folder a plugin keeps its data in, inside the store of a vault.
type PluginData struct {
	Options Options
}

// Open opens the store of the vault, answering for the folder named and no other.
func (p PluginData) Open(v domain.Vault, folder string) (port.PluginDataStore, error) {
	if folder == "" || strings.ContainsAny(folder, `/\.`) {
		return nil, fmt.Errorf("%q is not a folder of the store: %w", folder, ErrOutside)
	}
	store, err := OpenDerived(v.Path, p.Options, folder)
	if err != nil {
		return nil, err
	}
	return pluginFolder{store: store, folder: folder}, nil
}

// pluginFolder is one folder of a vault's store, whose files are named without it.
type pluginFolder struct {
	store  *DerivedStore
	folder string
}

func (f pluginFolder) key(name string) (string, error) {
	if name == "" || strings.ContainsAny(name, `/\`) || strings.HasPrefix(name, ".") {
		return "", fmt.Errorf("%q is not a file of the folder: %w", name, ErrOutside)
	}
	return f.folder + "/" + name, nil
}

func (f pluginFolder) Read(ctx context.Context, name string) ([]byte, error) {
	key, err := f.key(name)
	if err != nil {
		return nil, err
	}
	return f.store.Read(ctx, key)
}

func (f pluginFolder) Append(ctx context.Context, name string, content []byte) error {
	key, err := f.key(name)
	if err != nil {
		return err
	}
	return f.store.Append(ctx, key, content)
}

func (f pluginFolder) List(ctx context.Context) ([]port.PluginDataFile, error) {
	held, err := f.store.List(ctx, f.folder)
	if err != nil {
		return nil, err
	}
	out := make([]port.PluginDataFile, 0, len(held))
	for _, one := range held {
		out = append(out, port.PluginDataFile{Name: strings.TrimPrefix(one.Name, f.folder+"/"), Size: one.Size})
	}
	return out, nil
}
