package playback

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"path"
	"sort"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/port/playbackport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// host knows three plugins: one that keeps copies of link notes, one that keeps none, and one
// switched off that would keep copies of link notes too.
type host struct {
	pluginport.PluginDiscoverer
	pluginport.PluginResolver
}

func (host) GetPluginInfos() []pluginport.PluginInfo {
	return []pluginport.PluginInfo{{ID: "org.example.off"}, {ID: "org.example.links"}, {ID: "org.example.other"}}
}

func (host) GetPlugin(id string) (pluginport.DiscoveredPlugin, bool) {
	switch id {
	case "org.example.links":
		return pluginport.DiscoveredPlugin{Manifest: pluginport.PluginManifest{
			ID: id, Playback: &pluginport.PluginPlayback{NoteTypes: []string{"link"}},
		}}, true
	case "org.example.other":
		return pluginport.DiscoveredPlugin{Manifest: pluginport.PluginManifest{ID: id}}, true
	}
	return pluginport.DiscoveredPlugin{}, false
}

// contentTypes are the types the host gives the names of the tests.
var contentTypes = map[string]domain.ContentType{
	".md":   "text/markdown",
	".mp3":  "audio/mpeg",
	".mp4":  "video/mp4",
	".json": "application/json",
}

func (host) GetContentType(name string) domain.ContentType {
	if found, ok := contentTypes[strings.ToLower(path.Ext(name))]; ok {
		return found
	}
	return domain.UnknownContentType
}

// folders holds each plugin's folder in memory.
type folders map[string]map[string][]byte

func (f folders) Open(_ domain.Vault, pluginID string) (pluginport.PluginStore, error) {
	if f[pluginID] == nil {
		f[pluginID] = map[string][]byte{}
	}
	return folder(f[pluginID]), nil
}

func (f folders) DeleteFolder(_ context.Context, _ domain.Vault, pluginID string) error {
	delete(f, pluginID)
	return nil
}

type folder map[string][]byte

func (f folder) Read(_ context.Context, name string) ([]byte, error) {
	content, ok := f[name]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return content, nil
}

func (f folder) Append(_ context.Context, name string, content []byte) error {
	f[name] = append(f[name], content...)
	return nil
}

func (f folder) Write(_ context.Context, name string, content []byte) error {
	f[name] = content
	return nil
}

func (f folder) Delete(_ context.Context, name string) error {
	delete(f, name)
	return nil
}

func (f folder) List(context.Context) ([]pluginport.PluginStoreFile, error) {
	var out []pluginport.PluginStoreFile
	for name, content := range f {
		out = append(out, pluginport.PluginStoreFile{Name: name, Size: int64(len(content))})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (f folder) Open(_ context.Context, name string) (io.ReadSeekCloser, int64, error) {
	content, ok := f[name]
	if !ok {
		return nil, 0, fs.ErrNotExist
	}
	return nopCloser{bytes.NewReader(content)}, int64(len(content)), nil
}

func (f folder) Rename(_ context.Context, from, to string) error {
	f[to] = f[from]
	delete(f, from)
	return nil
}

type nopCloser struct{ io.ReadSeeker }

func (nopCloser) Close() error { return nil }

const hash = "4f2a"

func TestOnlyANoteTypeAnEnabledPluginDeclaresIsCopied(t *testing.T) {
	copies := NewProxies(host{}, folders{})
	if !copies.IsCopyingNotes() || !copies.IsCopied("links/Tidal mills.MD", "link") {
		t.Error("a link note is not copied, and the plugin that keeps copies of link notes is enabled")
	}
	if copies.IsCopied("notes/plain.md", domain.TypeNote) {
		t.Error("a plain note is copied, and no plugin keeps copies of plain notes")
	}
	for _, path := range []string{"talks/lecture.mp3", "notes/plain.json", "README"} {
		if copies.IsCopied(path, "") {
			t.Errorf("%s is copied, and no enabled plugin keeps copies of it", path)
		}
	}
}

func TestTheCopyIsTheFileNamedByTheHashInAContainerThatPlays(t *testing.T) {
	stores := folders{"org.example.links": {
		hash + ".part":         []byte("still arriving"),
		hash + ".json":         []byte("{}"),
		"0000.mp4":             []byte("another link's copy"),
		hash + ".mp4":          []byte("the copy"),
		"org.example.other.md": []byte("not a copy"),
	}}
	found, err := NewProxies(host{}, stores).OpenProxy(t.Context(), domain.Vault{}, "links/a.md", "link", hash)
	if err != nil {
		t.Fatal(err)
	}
	defer testsupport.Close(t, found.Body)
	body, err := io.ReadAll(found.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "the copy" || found.Size != 8 || found.MediaType != "video/mp4" {
		t.Errorf("the copy is %q, %d bytes, played as %q", body, found.Size, found.MediaType)
	}
}

func TestALinkWithNoCopyHasNone(t *testing.T) {
	stores := folders{
		"org.example.links": {hash + ".part": []byte("still arriving")},
		"org.example.other": {hash + ".mp4": []byte("a plugin that keeps no copies")},
		"org.example.off":   {hash + ".mp4": []byte("a plugin switched off")},
	}
	_, err := NewProxies(host{}, stores).OpenProxy(t.Context(), domain.Vault{}, "links/a.md", "link", hash)
	if !errors.Is(err, playbackport.ErrNoProxy) {
		t.Errorf("a link with no copy opened as %v", err)
	}
}

func TestACopyOfATypeThatDoesNotPlayIsNone(t *testing.T) {
	stores := folders{"org.example.links": {hash + ".json": []byte("{}")}}
	_, err := NewProxies(host{}, stores).OpenProxy(t.Context(), domain.Vault{}, "links/a.md", "link", hash)
	if !errors.Is(err, playbackport.ErrNoProxy) {
		t.Errorf("a file of no audio or video type opened as %v", err)
	}
}
