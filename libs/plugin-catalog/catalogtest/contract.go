package catalogtest

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-catalog/catalog"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
)

// Factory returns a source that lists exactly the entries given. An entry in the state
// catalog.StateInstalled or catalog.StateDisabled is a plugin on disk in that state.
type Factory func(t *testing.T, entries []catalog.Entry) catalog.Source

// OnDiskEntries are entries of plugins on disk, one enabled and one disabled, with contributions.
func OnDiskEntries() []catalog.Entry {
	return []catalog.Entry{
		sampleEntry("org.example.alpha", "1.0.0", catalog.StateInstalled, "views", "alpha"),
		sampleEntry("org.example.beta", "2.1.0", catalog.StateDisabled, "readers", "beta"),
	}
}

// OfferedEntries are entries of plugins a source offers and that are not on disk.
func OfferedEntries() []catalog.Entry {
	return []catalog.Entry{
		sampleEntry("org.example.gamma", "0.3.0", catalog.StateAvailable, "tools", "gamma"),
		sampleEntry("org.example.delta", "4.0.0", catalog.StateAvailable, "views", "delta"),
	}
}

func sampleEntry(id, version string, state catalog.State, category, kind string) catalog.Entry {
	return catalog.Entry{
		ID:          id,
		Name:        "Plugin " + kind,
		Version:     version,
		Description: "The " + kind + " plugin",
		Category:    category,
		Contributes: manifest.Contributes{
			TabKinds: []manifest.TabKind{{Kind: kind, Title: "Tab " + kind, Icon: "layers"}},
			Feature:  kind + " feature",
			Category: category,
		},
		Source:          "",
		State:           state,
		Requirements:    manifest.Requirements{Targets: []string{"desktop.editor"}, Protocol: 1, AppVersion: manifest.AppVersionRange{Min: "", Max: ""}},
		Incompatibility: nil,
	}
}

// RunSourceContract checks, over at least two entries, that the sources the factory returns satisfy
// catalog.Source: they list every entry with the identity they were given, find an entry by id,
// report a missing id as catalog.ErrNotFound, list nothing without failing when they hold nothing,
// hand out entries the caller can change freely, and stop at a context that has ended.
func RunSourceContract(t *testing.T, factory Factory, entries []catalog.Entry) {
	t.Helper()
	checks := []struct {
		name  string
		check func(t *testing.T, factory Factory, entries []catalog.Entry)
	}{
		{"lists every entry with its identity and the name of the source", checkList},
		{"gets an entry by id", checkGet},
		{"reports an id it does not list as not found", checkNotFound},
		{"lists nothing without failing when it holds nothing", checkEmpty},
		{"hands out entries the caller may change", checkIsolation},
		{"stops at a context that has ended", checkCanceled},
	}
	for _, one := range checks {
		t.Run(one.name, func(t *testing.T) { one.check(t, factory, entries) })
	}
}

func checkList(t *testing.T, factory Factory, entries []catalog.Entry) {
	t.Helper()
	source := factory(t, entries)

	listed, err := source.List(t.Context())

	if err != nil {
		t.Fatalf("List: %v", err)
	}
	assertSameEntries(t, source, listed, entries)
}

func checkGet(t *testing.T, factory Factory, entries []catalog.Entry) {
	t.Helper()
	source := factory(t, entries)

	got, err := source.Get(t.Context(), entries[1].ID)

	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	assertSameEntries(t, source, []catalog.Entry{got}, entries[1:2])
}

func checkNotFound(t *testing.T, factory Factory, entries []catalog.Entry) {
	t.Helper()
	source := factory(t, entries)

	_, err := source.Get(t.Context(), "org.example.missing")

	if !errors.Is(err, catalog.ErrNotFound) {
		t.Errorf("Get of a missing id: error = %v, want catalog.ErrNotFound", err)
	}
}

func checkEmpty(t *testing.T, factory Factory, _ []catalog.Entry) {
	t.Helper()
	source := factory(t, nil)

	listed, err := source.List(t.Context())

	if err != nil || len(listed) != 0 {
		t.Errorf("List = %v, %v; want no entries and no error", listed, err)
	}
}

func checkIsolation(t *testing.T, factory Factory, entries []catalog.Entry) {
	t.Helper()
	source := factory(t, entries)
	first, err := source.List(t.Context())
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	first[0].Name = "changed"
	second, err := source.List(t.Context())

	if err != nil {
		t.Fatalf("List again: %v", err)
	}
	assertSameEntries(t, source, second, entries)
}

func checkCanceled(t *testing.T, factory Factory, entries []catalog.Entry) {
	t.Helper()
	source := factory(t, entries)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, listErr := source.List(ctx)
	_, getErr := source.Get(ctx, entries[0].ID)

	if !errors.Is(listErr, context.Canceled) || !errors.Is(getErr, context.Canceled) {
		t.Errorf("errors = %v, %v; want context.Canceled", listErr, getErr)
	}
}

// assertSameEntries compares the identity, the contributions and the state of the entries, in any order.
func assertSameEntries(t *testing.T, source catalog.Source, got, want []catalog.Entry) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%d entries, want %d: %+v", len(got), len(want), got)
	}
	for _, wanted := range want {
		index := slices.IndexFunc(got, func(entry catalog.Entry) bool { return entry.ID == wanted.ID })
		if index < 0 {
			t.Errorf("entry %s is missing", wanted.ID)
			continue
		}
		entry := got[index]
		if entry.Source != source.Name() {
			t.Errorf("%s: source = %q, want %q", wanted.ID, entry.Source, source.Name())
		}
		if entry.Name != wanted.Name || entry.Version != wanted.Version || entry.Description != wanted.Description ||
			entry.State != wanted.State || entry.Category != wanted.Category {
			t.Errorf("%s: entry = %+v, want %+v", wanted.ID, entry, wanted)
		}
		if !slices.Equal(entry.Contributes.TabKinds, wanted.Contributes.TabKinds) || entry.Contributes.Feature != wanted.Contributes.Feature {
			t.Errorf("%s: contributes = %+v, want %+v", wanted.ID, entry.Contributes, wanted.Contributes)
		}
		if !slices.Equal(entry.Requirements.Targets, wanted.Requirements.Targets) {
			t.Errorf("%s: targets = %v, want %v", wanted.ID, entry.Requirements.Targets, wanted.Requirements.Targets)
		}
	}
}
