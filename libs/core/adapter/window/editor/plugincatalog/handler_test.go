package plugincatalog_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/plugin-catalog/catalog"
	"github.com/jiva-studio/numen/libs/plugin-catalog/catalogtest"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"

	"github.com/jiva-studio/numen/libs/core/adapter/window/editor/plugincatalog"
)

func entry(id string, state catalog.State, kind string) catalog.Entry {
	return catalog.Entry{
		ID: id, Name: "Plugin " + id, Version: "1.0.0", Description: "d", Category: "views",
		Contributes: manifest.Contributes{
			TabKinds: []manifest.TabKind{{Kind: kind, Title: "Tab", Icon: "layers"}}, Feature: "a feature", Category: "views",
		},
		Source: "", State: state,
		Requirements:    manifest.Requirements{Targets: nil, Protocol: 0, AppVersion: manifest.AppVersionRange{Min: "", Max: ""}},
		Incompatibility: nil,
	}
}

func newHandler(t *testing.T, sources ...catalog.Source) *plugincatalog.Handler {
	t.Helper()
	service, err := catalog.NewService(catalog.Config{Target: "desktop.editor", AppVersion: "2026.10.0", Sources: sources, InstallDir: "", CacheDir: ""})
	if err != nil {
		t.Fatal(err)
	}
	return plugincatalog.NewHandler(service)
}

func TestTheListCarriesEveryEntryWithItsStateAndContributions(t *testing.T) {
	handler := newHandler(t, catalogtest.NewFakeSource("local",
		entry("org.example.on", catalog.StateInstalled, "on"),
		entry("org.example.off", catalog.StateDisabled, "off")),
		catalogtest.NewFakeSource("remote", entry("org.example.offered", catalog.StateAvailable, "offered")))

	got, err := handler.ListCatalog(t.Context(), connect.NewRequest(&v1.ListCatalogRequest{}))

	if err != nil {
		t.Fatal(err)
	}
	states := map[string]v1.CatalogState{}
	for _, one := range got.Msg.GetEntries() {
		states[one.GetId()] = one.GetState()
	}
	want := map[string]v1.CatalogState{
		"org.example.on":      v1.CatalogState_CATALOG_STATE_INSTALLED,
		"org.example.off":     v1.CatalogState_CATALOG_STATE_DISABLED,
		"org.example.offered": v1.CatalogState_CATALOG_STATE_AVAILABLE,
	}
	for id, state := range want {
		if states[id] != state {
			t.Errorf("%s: state = %v, want %v", id, states[id], state)
		}
	}
	first := got.Msg.GetEntries()[0]
	if first.GetContributes().GetFeature() != "a feature" || first.GetContributes().GetTabKinds()[0].GetIcon() != "layers" || first.GetSource() == "" {
		t.Errorf("the entry lost its contributions or its source: %v", first)
	}
}

func TestListCatalogReportsAFailedSourceWithTheEntriesOfTheOthers(t *testing.T) {
	broken := catalogtest.NewFakeSource("remote")
	broken.Err = errors.New("offline")
	handler := newHandler(t, broken, catalogtest.NewFakeSource("local", entry("org.example.on", catalog.StateInstalled, "on")))

	got, err := handler.ListCatalog(t.Context(), connect.NewRequest(&v1.ListCatalogRequest{}))

	if err != nil {
		t.Fatalf("a failed source failed the call: %v", err)
	}
	failures := got.Msg.GetSourceErrors()
	if len(got.Msg.GetEntries()) != 1 || len(failures) != 1 || failures[0].GetSource() != "remote" || failures[0].GetMessage() != "offline" {
		t.Errorf("response = %v", got.Msg)
	}
}

func TestListCatalogReportsWhyAPluginCannotRun(t *testing.T) {
	elsewhere := entry("org.example.cards", catalog.StateInstalled, "deck")
	elsewhere.Requirements.Targets = []string{"desktop.flashcards"}
	tooNew := entry("org.example.new", catalog.StateAvailable, "future")
	tooNew.Requirements.AppVersion.Min = "2027.1.0"
	handler := newHandler(t, catalogtest.NewFakeSource("local", elsewhere, tooNew))

	got, err := handler.ListCatalog(t.Context(), connect.NewRequest(&v1.ListCatalogRequest{}))

	if err != nil {
		t.Fatal(err)
	}
	reasons := map[string]v1.CatalogIncompatibility{}
	for _, one := range got.Msg.GetEntries() {
		reasons[one.GetId()] = one.GetIncompatibility()
		if one.GetIncompatibility() != v1.CatalogIncompatibility_CATALOG_INCOMPATIBILITY_UNSPECIFIED && one.GetIncompatibilityDetail() == "" {
			t.Errorf("%s is refused with no detail", one.GetId())
		}
	}
	if reasons["org.example.cards"] != v1.CatalogIncompatibility_CATALOG_INCOMPATIBILITY_TARGET ||
		reasons["org.example.new"] != v1.CatalogIncompatibility_CATALOG_INCOMPATIBILITY_APP_TOO_OLD {
		t.Errorf("reasons = %v", reasons)
	}
}

func TestOnePluginIsReturnedByIdOrAnswersNotFound(t *testing.T) {
	handler := newHandler(t, catalogtest.NewFakeSource("local", entry("org.example.on", catalog.StateInstalled, "on")))

	found, err := handler.GetCatalogEntry(t.Context(), connect.NewRequest(&v1.GetCatalogEntryRequest{Id: "org.example.on"}))
	missing, missingErr := handler.GetCatalogEntry(t.Context(), connect.NewRequest(&v1.GetCatalogEntryRequest{Id: "org.example.none"}))

	if err != nil || found.Msg.GetEntry().GetId() != "org.example.on" || found.Msg.GetError() != v1.CatalogErrorCode_CATALOG_ERROR_CODE_UNSPECIFIED {
		t.Errorf("found = %v, %v", found, err)
	}
	if missingErr != nil || missing.Msg.GetError() != v1.CatalogErrorCode_CATALOG_ERROR_CODE_NOT_FOUND || missing.Msg.GetEntry() != nil {
		t.Errorf("missing = %v, %v; want an answer with the not-found code and no entry", missing, missingErr)
	}
}

func TestAKindOfTabIsFoundOrAnswersNotFound(t *testing.T) {
	handler := newHandler(t, catalogtest.NewFakeSource("local", entry("org.example.on", catalog.StateInstalled, "on")))

	found, err := handler.FindCatalogEntryByTabKind(t.Context(), connect.NewRequest(&v1.FindCatalogEntryByTabKindRequest{Kind: "on"}))
	missing, missingErr := handler.FindCatalogEntryByTabKind(t.Context(), connect.NewRequest(&v1.FindCatalogEntryByTabKindRequest{Kind: "none"}))

	if err != nil || found.Msg.GetEntry().GetId() != "org.example.on" {
		t.Errorf("found = %v, %v", found, err)
	}
	if missingErr != nil || missing.Msg.GetError() != v1.CatalogErrorCode_CATALOG_ERROR_CODE_NOT_FOUND {
		t.Errorf("missing = %v, %v", missing, missingErr)
	}
}

func TestAFailedSourceIsReportedWithAnEntryThatWasFound(t *testing.T) {
	broken := catalogtest.NewFakeSource("remote")
	broken.Err = errors.New("offline")
	handler := newHandler(t, broken, catalogtest.NewFakeSource("local", entry("org.example.on", catalog.StateInstalled, "on")))

	got, err := handler.GetCatalogEntry(t.Context(), connect.NewRequest(&v1.GetCatalogEntryRequest{Id: "org.example.on"}))
	missing, missingErr := handler.GetCatalogEntry(t.Context(), connect.NewRequest(&v1.GetCatalogEntryRequest{Id: "org.example.none"}))

	if err != nil || got.Msg.GetEntry().GetId() != "org.example.on" || len(got.Msg.GetSourceErrors()) != 1 {
		t.Errorf("found = %v, %v", got, err)
	}
	if missingErr != nil || missing.Msg.GetError() != v1.CatalogErrorCode_CATALOG_ERROR_CODE_NOT_FOUND || len(missing.Msg.GetSourceErrors()) != 1 {
		t.Errorf("missing = %v, %v; want not found with the failed source", missing, missingErr)
	}
}

func TestACanceledCallIsAStatusCode(t *testing.T) {
	handler := newHandler(t, catalogtest.NewFakeSource("local"))
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, listErr := handler.ListCatalog(ctx, connect.NewRequest(&v1.ListCatalogRequest{}))
	_, getErr := handler.GetCatalogEntry(ctx, connect.NewRequest(&v1.GetCatalogEntryRequest{Id: "x"}))
	_, findErr := handler.FindCatalogEntryByTabKind(ctx, connect.NewRequest(&v1.FindCatalogEntryByTabKindRequest{Kind: "x"}))

	for _, err := range []error{listErr, getErr, findErr} {
		if connect.CodeOf(err) != connect.CodeCanceled {
			t.Errorf("error = %v, want the canceled code", err)
		}
	}
}
