// Package playback finds the copies the enabled plugins keep of the vault's files in their folders
// of the vault's store.
package playback

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	pathpkg "path"
	"strings"

	"github.com/jiva-studio/numen/libs/notes/domain"

	playbackport "github.com/jiva-studio/numen/libs/core/usecase/file/playbackcontract"
	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
)

// Host finds the enabled plugins and gives a file its content type by its name.
type Host interface {
	pluginport.PluginDiscoverer
	pluginport.PluginResolver
	GetContentType(name string) domain.ContentType
}

// Copies are the copies of the enabled plugins that declare playback.
type Copies struct {
	host   Host
	stores pluginport.PluginStores
}

// NewProxies reaches the plugins of the host and their folders in the vaults' stores.
func NewProxies(host Host, stores pluginport.PluginStores) Copies {
	return Copies{host: host, stores: stores}
}

// getKeepers is the ids of the enabled plugins that keep copies of files of the path's type or of
// notes of the note type, in the order of their ids.
func (c Copies) getKeepers(path string, noteType domain.NoteType) []string {
	contentType := c.host.GetContentType(path)
	var out []string
	for _, info := range c.host.GetPluginInfos() {
		found, ok := c.host.GetPlugin(info.ID)
		if ok && found.Manifest.Playback != nil && found.Manifest.Playback.IsCopying(contentType, noteType) {
			out = append(out, info.ID)
		}
	}
	return out
}

func (c Copies) IsCopied(path string, noteType domain.NoteType) bool {
	return len(c.getKeepers(path, noteType)) > 0
}

func (c Copies) IsCopyingNotes() bool {
	for _, info := range c.host.GetPluginInfos() {
		found, ok := c.host.GetPlugin(info.ID)
		if ok && found.Manifest.Playback != nil && found.Manifest.Playback.HasNoteTypes() {
			return true
		}
	}
	return false
}

// OpenProxy opens the first copy named `<hash><ext>`, where the extension is of an audio or video
// type, in the folder of a plugin that keeps copies of the path's type or of the note type.
func (c Copies) OpenProxy(ctx context.Context, v domain.Vault, path string, noteType domain.NoteType, hash string) (playbackport.Copy, error) {
	for _, pluginID := range c.getKeepers(path, noteType) {
		store, err := c.stores.Open(v, pluginID)
		if err != nil {
			return playbackport.Copy{}, err
		}
		found, err := c.openIn(ctx, store, hash)
		if !errors.Is(err, playbackport.ErrNoProxy) {
			return found, err
		}
	}
	return playbackport.Copy{}, playbackport.ErrNoProxy
}

// playedTypes are the masks of the types a copy is played as.
var playedTypes = []domain.ContentType{"audio/*", "video/*"}

// openIn opens the copy named by the hash in one plugin's folder, or returns ErrNoProxy.
func (c Copies) openIn(ctx context.Context, store pluginport.PluginStore, hash string) (playbackport.Copy, error) {
	files, err := store.List(ctx)
	if err != nil {
		return playbackport.Copy{}, fmt.Errorf("list the files of the folder: %w", err)
	}
	for _, one := range files {
		if strings.TrimSuffix(one.Name, pathpkg.Ext(one.Name)) != hash {
			continue
		}
		mediaType := c.host.GetContentType(one.Name)
		if !isPlayed(mediaType) {
			continue
		}
		body, size, err := store.Open(ctx, one.Name)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return playbackport.Copy{}, err
		}
		return playbackport.Copy{Body: body, Size: size, MediaType: string(mediaType)}, nil
	}
	return playbackport.Copy{}, playbackport.ErrNoProxy
}

func isPlayed(contentType domain.ContentType) bool {
	for _, mask := range playedTypes {
		if mask.Matches(contentType) {
			return true
		}
	}
	return false
}


// NoProxies is an installation where no plugin keeps a copy.
type NoProxies struct{}

func (NoProxies) IsCopyingNotes() bool { return false }

func (NoProxies) IsCopied(string, domain.NoteType) bool { return false }

func (NoProxies) OpenProxy(context.Context, domain.Vault, string, domain.NoteType, string) (playbackport.Copy, error) {
	return playbackport.Copy{}, playbackport.ErrNoProxy
}
