package derivedfile

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/textfile"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/text"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/kernel/port/textport"
)

// TextStores opens the version files of a vault, `.numen/<plugin id>/<name>`, as JSON Lines of segments.
// Decoded keeps the segments of the files read last; nil keeps none.
type TextStores struct {
	Options vaultconfig.Options
	Decoded *DecodedTexts
}

// Open is the version files of one vault.
func (s TextStores) Open(v domain.Vault) (textport.TextFiles, error) {
	return textFiles{vault: v, stores: PluginStores{Options: s.Options}, decoded: s.Decoded}, nil
}

type textFiles struct {
	vault   domain.Vault
	stores  PluginStores
	decoded *DecodedTexts
}

func (f textFiles) ReadTextFile(ctx context.Context, pluginID, name string) (text.File, error) {
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
	hash := domain.HashContent(raw)
	if segments, ok := f.decoded.get(hash); ok {
		return text.File{Segments: segments, TextHash: hash}, nil
	}
	segments, err := textfile.Decode(raw)
	if err != nil {
		return text.File{}, fmt.Errorf("%s/%s: %w", pluginID, name, err)
	}
	f.decoded.put(hash, segments)
	return text.File{Segments: segments, TextHash: hash}, nil
}

func (f textFiles) WriteTextFile(ctx context.Context, pluginID, name string, segments []domain.TextSegment) (string, error) {
	raw, err := textfile.Encode(segments)
	if err != nil {
		return "", err
	}
	store, err := f.stores.OpenWriter(f.vault, pluginID)
	if err != nil {
		return "", err
	}
	if err := store.Write(ctx, name, raw); err != nil {
		return "", err
	}
	return domain.HashContent(raw), nil
}

func (f textFiles) DeleteTextFile(ctx context.Context, pluginID, name string) error {
	store, err := f.stores.OpenWriter(f.vault, pluginID)
	if err != nil {
		return err
	}
	if err := store.Delete(ctx, name); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}
