package service_test

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/agents-tools-window/backend/service"
)

func getGuideTabs() []host.OpenTab {
	return []host.OpenTab{
		{ID: "tab-1", Kind: "reader:reader-pdf", Path: guidePath, Title: "Bird Guide", Place: &host.TabPlace{Page: 4, PageCount: 120}},
		{ID: "tab-2", Kind: "reader:reader-pdf", Path: guidePath, Title: "Bird Guide", Place: &host.TabPlace{Page: 88, PageCount: 120}},
		{ID: "tab-3", Kind: "note", Path: notePath, Title: "Gray Heron"},
	}
}

// tabActivateOutput is the result of window_tab_activate.
type tabActivateOutput struct {
	Tab tabOutput `json:"tab"`
}

func requireWindowNotMoved(t *testing.T, window *hosttest.Window) {
	t.Helper()
	if notes, files := window.GetFocusedNotes(), window.GetFocusRequests(); len(notes) != 0 || len(files) != 0 {
		t.Errorf("the window received a call to focus notes %v and files %+v", notes, files)
	}
}

func TestWindowTabActivateActivatesTheTabWithTheID(t *testing.T) {
	w := newWindowTools(t)
	w.window.SetOpenTabs(getGuideTabs(), "tab-1")

	var out tabActivateOutput
	if failure := w.call(t, "window_tab_activate", `{"tabId":"tab-2"}`, nil, &out); failure != "" {
		t.Fatal(failure)
	}
	if out.Tab.TabID != "tab-2" || out.Tab.Kind != "reader:reader-pdf" || out.Tab.Path != guidePath || out.Tab.Title != "Bird Guide" || !out.Tab.IsActive {
		t.Errorf("returned %+v, want tab-2 of the guide, active", out.Tab)
	}
	requirePlace(t, "the returned tab", out.Tab.Place, `{"page":88,"pageCount":120}`)
	if activated := w.window.GetActivatedTabs(); !slices.Equal(activated, []string{"tab-2"}) {
		t.Errorf("the window activated %v, want [tab-2]", activated)
	}
	if _, activeID, err := w.window.ReadOpenTabs(t.Context()); err != nil || activeID != "tab-2" {
		t.Errorf("the window reports %q (%v) as active, want tab-2", activeID, err)
	}
	requireWindowNotMoved(t, w.window)
}

func TestWindowTabActivateRejectsAnUnknownIDByName(t *testing.T) {
	w := newWindowTools(t)
	w.window.SetOpenTabs(getGuideTabs(), "tab-1")

	failure := w.call(t, "window_tab_activate", `{"tabId":"tab-9"}`, nil, nil)
	if failure == "" {
		t.Fatal("a tab id no tab has was activated")
	}
	if !strings.Contains(failure, `"tab-9"`) {
		t.Errorf("the error %q does not name the id tab-9", failure)
	}
	if activated := w.window.GetActivatedTabs(); len(activated) != 0 {
		t.Errorf("the window activated %v", activated)
	}
	_, activeID, err := w.window.ReadOpenTabs(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if activeID != "tab-1" {
		t.Errorf("the window reports %q as active, want tab-1", activeID)
	}
	requireWindowNotMoved(t, w.window)
}

func TestWindowTabActivateRejectsAMissingTabID(t *testing.T) {
	for name, arguments := range map[string]string{
		"no arguments":              `{}`,
		"an empty tab id":           `{"tabId":""}`,
		"a path in place of an id":  `{"path":"library/Bird Guide.pdf"}`,
		"a title in place of an id": `{"title":"Gray Heron"}`,
	} {
		t.Run(name, func(t *testing.T) {
			w := newWindowTools(t)
			w.window.SetOpenTabs(getGuideTabs(), "tab-1")
			if failure := w.call(t, "window_tab_activate", arguments, nil, nil); failure == "" {
				t.Error("the call returned without an error")
			}
			if activated := w.window.GetActivatedTabs(); len(activated) != 0 {
				t.Errorf("the window activated %v", activated)
			}
			requireWindowNotMoved(t, w.window)
		})
	}
}

func TestWindowTabActivateRejectsACallWithNoWindow(t *testing.T) {
	w := newWindowTools(t)
	w.window.SetOpenTabs(getGuideTabs(), "tab-1")
	w.window.SetNoWindow()
	if failure := w.call(t, "window_tab_activate", `{"tabId":"tab-2"}`, nil, nil); failure == "" {
		t.Error("a host with no window activated the tab")
	}
}

func TestWindowTabActivateSchemaRequiresOnlyTheTabID(t *testing.T) {
	w := newWindowTools(t)
	schema := getToolSchema(t, w.registry, "window_tab_activate", false)
	requireKeys(t, "the input schema of window_tab_activate", getSchemaProperties(t, schema), []string{"tabId"}, []string{"path", "title"})
	required, isList := schema["required"].([]any)
	if !isList {
		t.Fatalf("the input schema has no list of required keys: %v", schema["required"])
	}
	if !slices.Contains(required, any("tabId")) {
		t.Errorf("the input schema requires %v, want tabId", schema["required"])
	}
}

func TestWindowTabActivateDescriptionNamesWindowTabList(t *testing.T) {
	w := newWindowTools(t)
	description := w.getToolInfo(t, "window_tab_activate").GetDescription()
	for _, want := range []string{"window_tab_list", "asked", "changes what the window shows"} {
		if !strings.Contains(description, want) {
			t.Errorf("the description of window_tab_activate does not contain %q: %s", want, description)
		}
	}
	for _, absent := range []string{"path", "title"} {
		if strings.Contains(description, absent) {
			t.Errorf("the description of window_tab_activate contains %q: %s", absent, description)
		}
	}
}

// otherActiveWindow is a window that does not make the requested tab active: ActivateTab returns the tab
// and the id of the tab that stays active.
type otherActiveWindow struct {
	*hosttest.Window
	activeID string
}

func (w otherActiveWindow) ActivateTab(ctx context.Context, id string) (host.OpenTab, string, error) {
	open, _, err := w.ReadOpenTabs(ctx)
	if err != nil {
		return host.OpenTab{}, "", err
	}
	for _, tab := range open {
		if tab.ID == id {
			return tab, w.activeID, nil
		}
	}
	return host.OpenTab{}, "", host.ErrNotExist
}

func TestWindowTabActivateReturnsIsActiveFromTheWindow(t *testing.T) {
	vault := hosttest.New(testVault)
	window := hosttest.NewWindow(vault)
	window.SetOpenTabs(getGuideTabs(), "tab-1")
	w := windowTools{window: window, registry: newRegistry(t, service.Host{Vault: vault, Window: otherActiveWindow{Window: window, activeID: "tab-1"}})}

	var out tabActivateOutput
	if failure := w.call(t, "window_tab_activate", `{"tabId":"tab-2"}`, nil, &out); failure != "" {
		t.Fatal(failure)
	}
	if out.Tab.TabID != "tab-2" || out.Tab.IsActive {
		t.Errorf("returned %+v, want tab-2 with isActive false", out.Tab)
	}
}

// countingWindow is the window of the vault that counts the ActivateTab calls it receives.
type countingWindow struct {
	*hosttest.Window
	activateCalls *atomic.Int64
}

func (w countingWindow) ActivateTab(ctx context.Context, id string) (host.OpenTab, string, error) {
	w.activateCalls.Add(1)
	return w.Window.ActivateTab(ctx, id)
}

func newCountingTools(t *testing.T) (windowTools, *atomic.Int64) {
	t.Helper()
	vault := hosttest.New(testVault)
	window := hosttest.NewWindow(vault)
	tabs := append(getGuideTabs(), host.OpenTab{ID: "tab-4", Kind: "reader:reader-pdf", Path: guidePath, Title: "Bird Guide", Place: &host.TabPlace{Page: 2, PageCount: 120}})
	window.SetOpenTabs(tabs, "tab-1")
	calls := &atomic.Int64{}
	on := service.Host{Vault: vault, Window: countingWindow{Window: window, activateCalls: calls}}
	return windowTools{window: window, registry: newRegistry(t, on)}, calls
}

func TestWindowTabActivateOfAnUnknownIDActivatesNothing(t *testing.T) {
	w, calls := newCountingTools(t)
	failure := w.call(t, "window_tab_activate", `{"tabId":"tab-9"}`, nil, nil)
	if !strings.Contains(failure, `no open tab has id "tab-9"`) {
		t.Errorf("the error is %q, want no open tab has id \"tab-9\"", failure)
	}
	if n := calls.Load(); n != 0 {
		t.Errorf("the window received a call to activate a tab %d times, want none", n)
	}
}

func TestWindowTabActivateActivatesATabOnce(t *testing.T) {
	w, calls := newCountingTools(t)
	var out tabActivateOutput
	if failure := w.call(t, "window_tab_activate", `{"tabId":"tab-4"}`, nil, &out); failure != "" {
		t.Fatal(failure)
	}
	if n := calls.Load(); n != 1 || out.Tab.TabID != "tab-4" || !out.Tab.IsActive {
		t.Errorf("the window received %d times and returned %+v, want once and tab-4 active", n, out.Tab)
	}
	requirePlace(t, "the returned tab", out.Tab.Place, `{"page":2,"pageCount":120}`)
}

// lateWindow is a window that does not show the requested tab in time: ActivateTab returns the error
// the host sends when the window does not report the tab active within its bound.
type lateWindow struct {
	*hosttest.Window
}

func (w lateWindow) ActivateTab(_ context.Context, id string) (host.OpenTab, string, error) {
	return host.OpenTab{}, "", fmt.Errorf("activate tab %q: the window did not show the tab in time", id)
}

func TestWindowTabActivateRejectsATabNotShownInTime(t *testing.T) {
	vault := hosttest.New(testVault)
	window := hosttest.NewWindow(vault)
	window.SetOpenTabs(getGuideTabs(), "tab-1")
	w := windowTools{window: window, registry: newRegistry(t, service.Host{Vault: vault, Window: lateWindow{Window: window}})}

	failure := w.call(t, "window_tab_activate", `{"tabId":"tab-2"}`, nil, nil)
	if failure == "" {
		t.Fatal("a tab the window did not show returned as focused")
	}
	if !strings.Contains(failure, `"tab-2"`) || !strings.Contains(failure, "did not show") || !strings.Contains(failure, "in time") {
		t.Errorf("the tool error is %q, want plain text that the window did not show tab-2 in time", failure)
	}
}
