// Package playback finds the copies the enabled plugins keep of the vault's files in their folders
// of the vault's store.
package playback

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	pathpkg "path"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/playbackport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// Host finds the enabled plugins.
type Host interface {
	pluginport.PluginDiscoverer
	pluginport.PluginResolver
}

// mediaTypes are the containers a copy is kept in, each against the type it is played as.
var mediaTypes = map[string]string{
	".mp4":  "video/mp4",
	".m4a":  "audio/mp4",
	".webm": "video/webm",
	".mp3":  "audio/mpeg",
	".ogg":  "audio/ogg",
	".wav":  "audio/wav",
	".flac": "audio/flac",
}

// Copies are the copies of the enabled plugins that declare playback.
type Copies struct {
	host   Host
	stores pluginport.PluginStores
}

// NewCopies reaches the plugins of the host and their folders in the vaults' stores.
func NewCopies(host Host, stores pluginport.PluginStores) Copies {
	return Copies{host: host, stores: stores}
}

// getKeepers is the ids of the enabled plugins that keep copies of files of the path's type, in the
// order of their ids.
func (c Copies) getKeepers(path string) []string {
	ext := strings.ToLower(pathpkg.Ext(path))
	if ext == "" {
		return nil
	}
	var out []string
	for _, info := range c.host.GetPluginInfos() {
		found, ok := c.host.GetPlugin(info.ID)
		if !ok || found.Manifest.Playback == nil {
			continue
		}
		if slices.ContainsFunc(found.Manifest.Playback.Files, func(one string) bool { return strings.EqualFold(one, ext) }) {
			out = append(out, info.ID)
		}
	}
	return out
}

func (c Copies) IsCopied(path string) bool { return len(c.getKeepers(path)) > 0 }

// OpenCopy opens the first copy named `<hash><ext>`, of a container a copy is kept in, in the folder
// of a plugin that keeps copies of the path's type.
func (c Copies) OpenCopy(ctx context.Context, v domain.Vault, path, hash string) (playbackport.Copy, error) {
	for _, pluginID := range c.getKeepers(path) {
		store, err := c.stores.Open(v, pluginID)
		if err != nil {
			return playbackport.Copy{}, err
		}
		found, err := openIn(ctx, store, hash)
		if !errors.Is(err, playbackport.ErrNoCopy) {
			return found, err
		}
	}
	return playbackport.Copy{}, playbackport.ErrNoCopy
}

// openIn opens the copy named by the hash in one plugin's folder, or returns ErrNoCopy.
func openIn(ctx context.Context, store pluginport.PluginStore, hash string) (playbackport.Copy, error) {
	files, err := store.List(ctx)
	if err != nil {
		return playbackport.Copy{}, fmt.Errorf("list the files of the folder: %w", err)
	}
	for _, one := range files {
		ext := pathpkg.Ext(one.Name)
		mediaType, ok := mediaTypes[ext]
		if !ok || strings.TrimSuffix(one.Name, ext) != hash {
			continue
		}
		body, size, err := store.Open(ctx, one.Name)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return playbackport.Copy{}, err
		}
		return playbackport.Copy{Body: body, Size: size, MediaType: mediaType}, nil
	}
	return playbackport.Copy{}, playbackport.ErrNoCopy
}
