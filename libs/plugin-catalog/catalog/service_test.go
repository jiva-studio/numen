package catalog_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-catalog/catalog"
	"github.com/jiva-studio/numen/libs/plugin-catalog/catalogtest"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
)

const (
	editor     = "desktop.editor"
	flashcards = "desktop.flashcards"
)

func entry(id, version string, state catalog.State, kinds ...string) catalog.Entry {
	tabs := make([]manifest.TabKind, 0, len(kinds))
	for _, kind := range kinds {
		tabs = append(tabs, manifest.TabKind{Kind: kind, Title: kind, Icon: ""})
	}
	return catalog.Entry{
		ID: id, Name: id, Version: version, Description: "", Category: "",
		Contributes:     manifest.Contributes{TabKinds: tabs, Feature: "", Category: ""},
		Source:          "",
		State:           state,
		Requirements:    manifest.Requirements{Targets: nil, Protocol: 0, AppVersion: manifest.AppVersionRange{Min: "", Max: ""}},
		Incompatibility: nil,
	}
}

func newService(t *testing.T, sources ...catalog.Source) *catalog.Service {
	t.Helper()
	service, err := catalog.NewService(catalog.Config{Target: editor, AppVersion: "2026.10.0", Sources: sources, InstallDir: "", CacheDir: ""})
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func list(t *testing.T, service *catalog.Service) catalog.Listing {
	t.Helper()
	listing, err := service.List(t.Context())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	return listing
}

func find(listing catalog.Listing, id string) (catalog.Entry, bool) {
	for _, one := range listing.Entries {
		if one.ID == id {
			return one, true
		}
	}
	return catalog.Entry{}, false
}

func TestAServiceNeedsATarget(t *testing.T) {
	if _, err := catalog.NewService(catalog.Config{Target: " ", AppVersion: "", Sources: nil, InstallDir: "", CacheDir: ""}); err == nil {
		t.Error("a config with no target was accepted")
	}
}

func TestTheSameIdFromTwoSourcesIsOneEntry(t *testing.T) {
	local := catalogtest.NewFakeSource("local", entry("org.example.a", "1.0.0", catalog.StateInstalled))
	remote := catalogtest.NewFakeSource("remote",
		entry("org.example.a", "2.0.0", catalog.StateAvailable),
		entry("org.example.b", "1.0.0", catalog.StateAvailable))

	listing := list(t, newService(t, remote, local))

	if len(listing.Entries) != 2 {
		t.Fatalf("%d entries, want 2: %+v", len(listing.Entries), listing.Entries)
	}
	a, _ := find(listing, "org.example.a")
	if a.Source != "local" || a.State != catalog.StateInstalled || a.Version != "1.0.0" {
		t.Errorf("the installed entry did not win over the newer remote one: %+v", a)
	}
	if b, _ := find(listing, "org.example.b"); b.Source != "remote" {
		t.Errorf("the entry only the remote source lists: %+v", b)
	}
}

func TestOfferedEntriesOfOneIdTakeTheNewerVersionAndATieGoesToTheFirstSource(t *testing.T) {
	first := catalogtest.NewFakeSource("first",
		entry("org.example.a", "1.0.0", catalog.StateAvailable),
		entry("org.example.b", "1.0.0", catalog.StateAvailable))
	second := catalogtest.NewFakeSource("second",
		entry("org.example.a", "1.2.0", catalog.StateAvailable),
		entry("org.example.b", "1.0.0", catalog.StateAvailable))

	listing := list(t, newService(t, first, second))

	if a, _ := find(listing, "org.example.a"); a.Source != "second" {
		t.Errorf("the newer version lost: %+v", a)
	}
	if b, _ := find(listing, "org.example.b"); b.Source != "first" {
		t.Errorf("a tie did not go to the first source: %+v", b)
	}
}

func TestADisabledPluginIsListedAndOutranksTheOneOnOffer(t *testing.T) {
	local := catalogtest.NewFakeSource("local", entry("org.example.a", "1.0.0", catalog.StateDisabled))
	remote := catalogtest.NewFakeSource("remote", entry("org.example.a", "9.0.0", catalog.StateAvailable))

	listing := list(t, newService(t, remote, local))

	if a, ok := find(listing, "org.example.a"); !ok || a.State != catalog.StateDisabled {
		t.Errorf("entry = %+v, %v; want the disabled plugin", a, ok)
	}
}

func TestASourceThatFailsDoesNotDropTheOthers(t *testing.T) {
	broken := catalogtest.NewFakeSource("remote")
	broken.Err = errors.New("offline")
	local := catalogtest.NewFakeSource("local", entry("org.example.a", "1.0.0", catalog.StateInstalled))

	listing, err := newService(t, broken, local).List(t.Context())

	var failure *catalog.SourceError
	if !errors.As(err, &failure) || failure.Source != "remote" || !errors.Is(err, broken.Err) {
		t.Errorf("error = %v, want the failure of the remote source", err)
	}
	if len(listing.Errors) != 1 || len(listing.Entries) != 1 || listing.Entries[0].ID != "org.example.a" {
		t.Errorf("listing = %+v, want the entry of the source that answered and one error", listing)
	}
}

func TestASourceThatReadPartOfItsPlaceKeepsItsEntries(t *testing.T) {
	partial := catalogtest.NewFakeSource("local", entry("org.example.a", "1.0.0", catalog.StateInstalled))
	partial.Err = errors.New("one folder could not be read")

	listing, err := newService(t, partial).List(t.Context())

	if err == nil || len(listing.Entries) != 1 {
		t.Errorf("listing = %+v, error = %v; want the entry with the error", listing, err)
	}
}

func TestACanceledContextStopsTheListing(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := newService(t, catalogtest.NewFakeSource("local")).List(ctx)

	if !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want context.Canceled", err)
	}
}

func TestGetFindsOnePluginOrReportsNotFound(t *testing.T) {
	service := newService(t, catalogtest.NewFakeSource("local", entry("org.example.a", "1.0.0", catalog.StateInstalled)))

	got, err := service.Get(t.Context(), "org.example.a")
	_, missing := service.Get(t.Context(), "org.example.b")

	if err != nil || got.ID != "org.example.a" {
		t.Errorf("Get = %+v, %v", got, err)
	}
	if !errors.Is(missing, catalog.ErrNotFound) {
		t.Errorf("Get of a missing plugin: error = %v, want ErrNotFound", missing)
	}
}

func TestAKindOfTabIsFoundInThePluginThatProvidesIt(t *testing.T) {
	service := newService(t,
		catalogtest.NewFakeSource("remote", entry("org.example.offered", "1.0.0", catalog.StateAvailable, "plex")),
		catalogtest.NewFakeSource("local", entry("org.example.disk", "1.0.0", catalog.StateDisabled, "plex", "other")))

	got, err := service.FindByTabKind(t.Context(), "plex")
	_, missing := service.FindByTabKind(t.Context(), "nothing")

	if err != nil || got.ID != "org.example.disk" {
		t.Errorf("FindByTabKind = %+v, %v; want the plugin on disk", got, err)
	}
	if !errors.Is(missing, catalog.ErrNotFound) {
		t.Errorf("a kind no plugin provides: error = %v, want ErrNotFound", missing)
	}
}

func TestAnIncompatiblePluginIsReportedWithItsReason(t *testing.T) {
	tooNew := entry("org.example.new", "1.0.0", catalog.StateAvailable, "future")
	tooNew.Requirements.AppVersion.Min = "2027.1.0"
	elsewhere := entry("org.example.cards", "1.0.0", catalog.StateAvailable, "deck")
	elsewhere.Requirements.Targets = []string{flashcards}
	fine := entry("org.example.fine", "1.0.0", catalog.StateAvailable, "fine")
	service := newService(t, catalogtest.NewFakeSource("remote", tooNew, elsewhere, fine))

	listing := list(t, service)
	_, installTooNew := service.PrepareInstall(t.Context(), "org.example.new")
	_, installElsewhere := service.PrepareInstall(t.Context(), "org.example.cards")
	prepared, installFine := service.PrepareInstall(t.Context(), "org.example.fine")
	_, findIncompatible := service.FindByTabKind(t.Context(), "future")

	if listed, _ := find(listing, "org.example.new"); listed.Incompatibility == nil || listed.Incompatibility.Reason != manifest.ReasonAppTooOld {
		t.Errorf("the listing does not say why the plugin is refused: %+v", listed.Incompatibility)
	}
	assertRefusal(t, installTooNew, manifest.ReasonAppTooOld)
	assertRefusal(t, installElsewhere, manifest.ReasonTarget)
	if installFine != nil || prepared.ID != "org.example.fine" {
		t.Errorf("PrepareInstall of a compatible plugin = %+v, %v", prepared, installFine)
	}
	if !errors.Is(findIncompatible, catalog.ErrNotFound) {
		t.Errorf("a plugin the application cannot run provides its tab kind: error = %v", findIncompatible)
	}
}

func assertRefusal(t *testing.T, err error, reason manifest.IncompatibleReason) {
	t.Helper()
	var incompatible *manifest.IncompatibleError
	if !errors.As(err, &incompatible) || incompatible.Reason != reason || incompatible.Detail == "" {
		t.Errorf("error = %v, want a refusal for %q with a detail", err, reason)
	}
}

func TestAPluginOnDiskIsNotInstalledAgain(t *testing.T) {
	service := newService(t, catalogtest.NewFakeSource("local", entry("org.example.a", "1.0.0", catalog.StateDisabled)))

	_, err := service.PrepareInstall(t.Context(), "org.example.a")
	_, missing := service.PrepareInstall(t.Context(), "org.example.b")

	if !errors.Is(err, catalog.ErrAlreadyOnDisk) {
		t.Errorf("error = %v, want ErrAlreadyOnDisk", err)
	}
	if !errors.Is(missing, catalog.ErrNotFound) {
		t.Errorf("error = %v, want ErrNotFound", missing)
	}
}

func TestTheFakeSourceSatisfiesTheSourceContract(t *testing.T) {
	catalogtest.RunSourceContract(t, func(_ *testing.T, entries []catalog.Entry) catalog.Source {
		return catalogtest.NewFakeSource("fake", entries...)
	}, append(catalogtest.OnDiskEntries(), catalogtest.OfferedEntries()...))
}
