// Package hintsource offers the plugins that ship with the application when none of them is on
// disk, so that a tab or a file whose plugin is missing can say which plugin provides it before any
// online source is read.
package hintsource

import (
	"context"
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-catalog/catalog"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
)

// Name is the name of the source in its entries.
const Name = "hint"

// Hint is a plugin the application knows without reading its manifest.
type Hint struct {
	ID          string
	Name        string
	Description string
	Contributes manifest.Contributes
	// Extensions are the extensions, lowercase and without the dot, of the files the plugin opens.
	Extensions []string
	// FileNames are the names of the files the plugin opens whatever their extension.
	FileNames []string
}

// Source lists hints as available plugins.
type Source struct {
	hints []Hint
}

// New returns the source over the hints.
func New(hints []Hint) *Source {
	return &Source{hints: slices.Clone(hints)}
}

// NewBuiltIn returns the source over the plugins that ship with the application.
func NewBuiltIn() *Source {
	return New(BuiltIn())
}

// Name is Name.
func (s *Source) Name() string { return Name }

// List returns an available entry for each hint. A hint states no version, so the entry has none.
func (s *Source) List(ctx context.Context) ([]catalog.Entry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	entries := make([]catalog.Entry, 0, len(s.hints))
	for _, hint := range s.hints {
		entries = append(entries, newEntry(hint))
	}
	return entries, nil
}

// Get returns the entry of the hint of the plugin.
func (s *Source) Get(ctx context.Context, id string) (catalog.Entry, error) {
	if err := ctx.Err(); err != nil {
		return catalog.Entry{}, err
	}
	for _, hint := range s.hints {
		if hint.ID == id {
			return newEntry(hint), nil
		}
	}
	return catalog.Entry{}, fmt.Errorf("plugin %q: %w", id, catalog.ErrNotFound)
}

// FindByFile returns the entry of the plugin that opens a file of the name, and false when no hint
// names it. The name is matched without regard to case.
func (s *Source) FindByFile(name string) (catalog.Entry, bool) {
	base := strings.ToLower(path.Base(strings.ReplaceAll(name, `\`, "/")))
	extension := strings.TrimPrefix(path.Ext(base), ".")
	for _, hint := range s.hints {
		if slices.Contains(hint.FileNames, base) || (extension != "" && slices.Contains(hint.Extensions, extension)) {
			return newEntry(hint), true
		}
	}
	return catalog.Entry{}, false
}

func newEntry(hint Hint) catalog.Entry {
	return catalog.Entry{
		ID:              hint.ID,
		Name:            hint.Name,
		Version:         "",
		Description:     hint.Description,
		Category:        hint.Contributes.Category,
		Contributes:     hint.Contributes,
		Source:          Name,
		State:           catalog.StateAvailable,
		Requirements:    manifest.Requirements{Targets: nil, Protocol: 0, AppVersion: manifest.AppVersionRange{Min: "", Max: ""}},
		Incompatibility: nil,
	}
}
