// Package indexsource offers the plugins of the plugin index as available plugins, so that a tab or
// a file whose plugin is not on disk can name the plugin that provides it before any online source
// is read.
//
// The index is the file FileName, generated at build time from the manifest of every plugin of the
// repository: a JSON object whose "plugins" member lists the manifests. Where no index exists, the
// source reads the manifests of the fallback folders.
package indexsource

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"slices"

	"github.com/jiva-studio/numen/libs/plugin/catalog/catalog"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
)

// Name is the name of the source in its entries.
const Name = "index"

const FileName = "plugins-index.json"

// Source lists the plugins of the index as available plugins.
type Source struct {
	indexPaths   []string
	manifestDirs []string
}

// New returns the source over the first of the index files that exists. When none exists, it reads
// the plugin folders under manifestDirs.
func New(indexPaths, manifestDirs []string) *Source {
	return &Source{indexPaths: slices.Clone(indexPaths), manifestDirs: slices.Clone(manifestDirs)}
}

func (s *Source) Name() string { return Name }

// List returns an available entry for each manifest of the index. A manifest that is invalid is
// reported in the error, which joins one error for each, and does not hide the manifests beside it.
func (s *Source) List(ctx context.Context) ([]catalog.Entry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	manifests, err := s.readManifests()
	entries := make([]catalog.Entry, 0, len(manifests))
	for _, one := range manifests {
		entries = append(entries, catalog.NewEntry(Name, catalog.StateAvailable, one))
	}
	return entries, err
}

// Get returns the entry of the plugin. A plugin that is not in the index is catalog.ErrNotFound.
func (s *Source) Get(ctx context.Context, id string) (catalog.Entry, error) {
	entries, err := s.List(ctx)
	// gomutants:disable-next-line BRANCH_IF reason="List already returns empty entries when ctx is canceled"
	if ctxErr := ctx.Err(); ctxErr != nil {
		return catalog.Entry{}, ctxErr
	}
	for _, entry := range entries {
		if entry.ID == id {
			return entry, err
		}
	}
	return catalog.Entry{}, errors.Join(fmt.Errorf("plugin %q: %w", id, catalog.ErrNotFound), err)
}

func (s *Source) readManifests() ([]manifest.Manifest, error) {
	for _, path := range s.indexPaths {
		data, err := os.ReadFile(path)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read the plugin index: %w", err)
		}
		return decodeIndex(path, data)
	}
	return s.discoverManifests()
}

func (s *Source) discoverManifests() ([]manifest.Manifest, error) {
	found, problems := manifestjson.Discover(s.manifestDirs)
	manifests := make([]manifest.Manifest, 0, len(found))
	for _, plugin := range found {
		manifests = append(manifests, plugin.Manifest)
	}
	failures := make([]error, 0, len(problems))
	for _, problem := range problems {
		failures = append(failures, fmt.Errorf("%s: %w", problem.Dir, problem.Err))
	}
	return manifests, errors.Join(failures...)
}

func decodeIndex(path string, data []byte) ([]manifest.Manifest, error) {
	var index struct {
		Plugins []json.RawMessage `json:"plugins"`
	}
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("decode the plugin index %s: %w", path, err)
	}
	manifests := make([]manifest.Manifest, 0, len(index.Plugins))
	var failures []error
	for position, raw := range index.Plugins {
		one, err := manifestjson.Read(bytes.NewReader(raw))
		if err != nil {
			failures = append(failures, fmt.Errorf("plugin %d of the index %s: %w", position, path, err))
			continue
		}
		manifests = append(manifests, one)
	}
	return manifests, errors.Join(failures...)
}
