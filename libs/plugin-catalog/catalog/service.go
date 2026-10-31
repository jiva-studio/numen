package catalog

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
)

// Config is what an application hands the catalog.
type Config struct {
	// Target is the window the application opens, as plugins name it in their manifests.
	Target string
	// AppVersion is the version of the application. Empty is an application that states none, and no
	// bound on the versions applies.
	AppVersion string
	// Sources are the places plugins are found in, in the order that decides between two entries that
	// are equal in every other way.
	Sources []Source
	// InstallDir is the folder installed plugins are unpacked into.
	InstallDir string
	// CacheDir is the folder downloaded packages are kept in.
	CacheDir string
}

// Listing is the merged entries of the sources, and the sources that could not be read in full.
type Listing struct {
	// Entries are ordered by plugin id.
	Entries []Entry
	Errors  []*SourceError
}

// Service lists the plugins of every source and finds one among them.
type Service struct {
	sources []Source
	env     manifest.Environment
}

var errNoTarget = errors.New("the catalog config names no target")

// NewService returns the service over the sources of the config.
func NewService(config Config) (*Service, error) {
	if strings.TrimSpace(config.Target) == "" {
		return nil, errNoTarget
	}
	return &Service{
		sources: slices.Clone(config.Sources),
		env:     manifest.Environment{Target: config.Target, AppVersion: config.AppVersion},
	}, nil
}

// List asks every source, merges the entries by plugin id and marks the entries the application
// cannot run. A source that fails does not remove the entries of the others: the error returned
// joins one *SourceError for each source that failed, and the listing holds what the rest returned.
func (s *Service) List(ctx context.Context) (Listing, error) {
	lists := make([][]Entry, 0, len(s.sources))
	var failures []*SourceError
	for _, source := range s.sources {
		if err := ctx.Err(); err != nil {
			return Listing{}, err
		}
		entries, err := source.List(ctx)
		if err != nil {
			failures = append(failures, &SourceError{Source: source.Name(), Err: err})
		}
		lists = append(lists, entries)
	}
	merged := mergeEntries(lists)
	for i := range merged {
		merged[i].Incompatibility = checkEntry(merged[i], s.env)
	}
	listing := Listing{Entries: merged, Errors: failures}
	return listing, joinSourceErrors(failures)
}

func joinSourceErrors(failures []*SourceError) error {
	errs := make([]error, 0, len(failures))
	for _, failure := range failures {
		errs = append(errs, failure)
	}
	return errors.Join(errs...)
}

func checkEntry(entry Entry, env manifest.Environment) *manifest.IncompatibleError {
	var incompatible *manifest.IncompatibleError
	if errors.As(entry.Requirements.Check(env), &incompatible) {
		return incompatible
	}
	return nil
}

// Get returns the entry of the plugin after the merge. The error is ErrNotFound when no source lists
// it, and the sources that failed are joined to it.
func (s *Service) Get(ctx context.Context, id string) (Entry, error) {
	listing, err := s.List(ctx)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return Entry{}, ctxErr
	}
	for _, entry := range listing.Entries {
		if entry.ID == id {
			return entry, err
		}
	}
	return Entry{}, errors.Join(fmt.Errorf("plugin %q: %w", id, ErrNotFound), err)
}

// FindByTabKind returns the entry of the plugin that provides the kind of tab. A plugin the
// application cannot run does not provide it, and a plugin on disk is preferred to one on offer.
func (s *Service) FindByTabKind(ctx context.Context, kind string) (Entry, error) {
	listing, err := s.List(ctx)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return Entry{}, ctxErr
	}
	var found *Entry
	for i := range listing.Entries {
		entry := &listing.Entries[i]
		if entry.Incompatibility != nil || !entry.Contributes.HasTabKind(kind) {
			continue
		}
		if found == nil || (entry.State.IsOnDisk() && !found.State.IsOnDisk()) {
			found = entry
		}
	}
	if found == nil {
		return Entry{}, errors.Join(fmt.Errorf("tab kind %q: %w", kind, ErrNotFound), err)
	}
	return *found, err
}

// PrepareInstall returns the entry of an available plugin that the application can run. A plugin
// that is on disk is ErrAlreadyOnDisk, and one the application cannot run is refused with the
// *manifest.IncompatibleError that says why.
func (s *Service) PrepareInstall(ctx context.Context, id string) (Entry, error) {
	entry, err := s.Get(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return Entry{}, err
	}
	switch {
	case entry.State.IsOnDisk():
		return Entry{}, fmt.Errorf("plugin %q: %w", id, ErrAlreadyOnDisk)
	case entry.Incompatibility != nil:
		return Entry{}, fmt.Errorf("plugin %q: %w", id, entry.Incompatibility)
	}
	return entry, nil
}
