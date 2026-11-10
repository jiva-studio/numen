// Package textfiles reads and writes the version files in the plugin folders
// of a vault's store, `.numen/<plugin id>/<name>`, as JSON Lines of segments.
package textfiles

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/textfile"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/text"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/derivedfile"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/kernel/port/textport"
)

// Stores opens the version files of a vault.
type Stores struct {
	Options vaultconfig.Options
}

// Open is the version files of one vault.
func (s Stores) Open(v domain.Vault) (textport.TextFiles, error) {
	return files{vault: v, stores: derivedfile.PluginStores{Options: s.Options}}, nil
}

type files struct {
	vault  domain.Vault
	stores derivedfile.PluginStores
}

func (f files) ReadTextFile(ctx context.Context, pluginID, name string) (text.File, error) {
	store, err := f.stores.Open(f.vault, pluginID)
	if err != nil {
		return text.File{}, err
	}
	raw, err := store.Read(ctx, name)
	if errors.Is(err, fs.ErrNotExist) {
		return text.File{}, fmt.Errorf("%w: %s/%s", text.ErrNoText, pluginID, name)
	}
	if err != nil {
		return text.File{}, err
	}
	segments, err := textfile.Decode(raw)
	if err != nil {
		return text.File{}, fmt.Errorf("%s/%s: %w", pluginID, name, err)
	}
	return text.File{Segments: segments, TextHash: domain.HashContent(raw)}, nil
}

func (f files) WriteTextFile(ctx context.Context, pluginID, name string, segments []domain.TextSegment) (string, error) {
	raw, err := textfile.Encode(segments)
	if err != nil {
		return "", err
	}
	store, err := f.stores.Open(f.vault, pluginID)
	if err != nil {
		return "", err
	}
	if err := store.Write(ctx, name, raw); err != nil {
		return "", err
	}
	return domain.HashContent(raw), nil
}

func (f files) DeleteTextFile(ctx context.Context, pluginID, name string) error {
	store, err := f.stores.Open(f.vault, pluginID)
	if err != nil {
		return err
	}
	if err := store.Delete(ctx, name); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}
