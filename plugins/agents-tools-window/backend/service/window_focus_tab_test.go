package service_test

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/agents-tools-window/backend/service"
)

// getGuideTabs is one bird guide open in two tabs, and a note.
func getGuideTabs() []host.OpenTab {
	return []host.OpenTab{
		{ID: "tab-1", Kind: "reader:reader-pdf", Path: guidePath, Title: "Bird Guide", State: `{"page":4,"pageCount":120}`},
		{ID: "tab-2", Kind: "reader:reader-pdf", Path: guidePath, Title: "Bird Guide", State: `{"page":88,"pageCount":120}`},
		{ID: "tab-3", Kind: "note", Path: notePath, Title: "Grey Heron"},
	}
}

// focusTabOutput is what window_focus_tab answers.
type focusTabOutput struct {
	Tab tabOutput `json:"tab"`
}

// requireWindowNotMoved fails the test when the window was asked to focus a note or a file.
func requireWindowNotMoved(t *testing.T, window *hosttest.Window) {
	t.Helper()
	if notes, files := window.GetFocusedNotes(), window.GetFocusRequests(); len(notes) != 0 || len(files) != 0 {
		t.Errorf("the window was asked to focus notes %v and files %+v", notes, files)
	}
}

func TestWindowFocusTabActivatesTheTabWithTheIDAmongTabsOfOneFile(t *testing.T) {
	w := newWindowTools(t)
	w.window.SetOpenTabs(getGuideTabs(), "tab-1")

	var out focusTabOutput
	if failure := w.call(t, "window_focus_tab", `{"tabId":"tab-2"}`, nil, &out); failure != "" {
		t.Fatal(failure)
	}
	if out.Tab.ID != "tab-2" || out.Tab.Kind != "reader:reader-pdf" || out.Tab.Path != guidePath || out.Tab.Title != "Bird Guide" || !out.Tab.IsActive {
		t.Errorf("answered %+v, want tab-2 of the guide, active", out.Tab)
	}
	requireLocation(t, "the answered tab", out.Tab.Location, `{"page":88,"pageCount":120}`)
	if activated := w.window.GetActivatedTabs(); !slices.Equal(activated, []string{"tab-2"}) {
		t.Errorf("the window activated %v, want [tab-2]", activated)
	}
	if _, activeID, err := w.window.ReadOpenTabs(t.Context()); err != nil || activeID != "tab-2" {
		t.Errorf("the window reports %q (%v) as active, want tab-2", activeID, err)
	}
	requireWindowNotMoved(t, w.window)
}

func TestWindowFocusTabOfAnIDNoTabHasIsAToolErrorNamingTheID(t *testing.T) {
	w := newWindowTools(t)
	w.window.SetOpenTabs(getGuideTabs(), "tab-1")

	failure := w.call(t, "window_focus_tab", `{"tabId":"tab-9"}`, nil, nil)
	if failure == "" {
		t.Fatal("a tab id no tab has was activated")
	}
	if !strings.Contains(failure, `"tab-9"`) {
		t.Errorf("the error %q does not name the id tab-9", failure)
	}
	if activated := w.window.GetActivatedTabs(); len(activated) != 0 {
		t.Errorf("the window activated %v", activated)
	}
	if _, activeID, _ := w.window.ReadOpenTabs(t.Context()); activeID != "tab-1" {
		t.Errorf("the window reports %q as active, want tab-1", activeID)
	}
	requireWindowNotMoved(t, w.window)
}

func TestWindowFocusTabWithoutATabIDIsAToolError(t *testing.T) {
	for name, arguments := range map[string]string{
		"no arguments":              `{}`,
		"an empty tab id":           `{"tabId":""}`,
		"a path in place of an id":  `{"path":"library/Bird Guide.pdf"}`,
		"a title in place of an id": `{"title":"Grey Heron"}`,
	} {
		t.Run(name, func(t *testing.T) {
			w := newWindowTools(t)
			w.window.SetOpenTabs(getGuideTabs(), "tab-1")
			if failure := w.call(t, "window_focus_tab", arguments, nil, nil); failure == "" {
				t.Error("the call was answered without an error")
			}
			if activated := w.window.GetActivatedTabs(); len(activated) != 0 {
				t.Errorf("the window activated %v", activated)
			}
			requireWindowNotMoved(t, w.window)
		})
	}
}

func TestWindowFocusTabWithNoWindowIsAToolError(t *testing.T) {
	w := newWindowTools(t)
	w.window.SetOpenTabs(getGuideTabs(), "tab-1")
	w.window.SetNoWindow()
	if failure := w.call(t, "window_focus_tab", `{"tabId":"tab-2"}`, nil, nil); failure == "" {
		t.Error("a host with no window activated the tab")
	}
}

func TestWindowFocusTabInputSchemaRequiresTheTabIDAlone(t *testing.T) {
	w := newWindowTools(t)
	schema := getToolSchema(t, w.calls, "window_focus_tab", false)
	requireKeys(t, "the input schema of window_focus_tab", getSchemaProperties(t, schema), []string{"tabId"}, []string{"path", "title"})
	required, _ := schema["required"].([]any)
	if !slices.Contains(required, any("tabId")) {
		t.Errorf("the input schema requires %v, want tabId", schema["required"])
	}
}

func TestWindowFocusTabDescriptionNamesWindowTabsGet(t *testing.T) {
	w := newWindowTools(t)
	description := w.getToolInfo(t, "window_focus_tab").GetDescription()
	for _, want := range []string{"window_tabs_get", "asked", "changes what the window shows"} {
		if !strings.Contains(description, want) {
			t.Errorf("the description of window_focus_tab does not say %q: %s", want, description)
		}
	}
	for _, absent := range []string{"path", "title"} {
		if strings.Contains(description, absent) {
			t.Errorf("the description of window_focus_tab says %q: %s", absent, description)
		}
	}
}

// otherActiveWindow is a window that does not make the asked tab active: ActivateTab answers the tab
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

func TestWindowFocusTabAnswersIsActiveFromTheWindow(t *testing.T) {
	vault := hosttest.New(testVault)
	window := hosttest.NewWindow(vault)
	window.SetOpenTabs(getGuideTabs(), "tab-1")
	w := windowTools{window: window, calls: newRegistry(t, service.Host{Vault: vault, Window: otherActiveWindow{Window: window, activeID: "tab-1"}})}

	var out focusTabOutput
	if failure := w.call(t, "window_focus_tab", `{"tabId":"tab-2"}`, nil, &out); failure != "" {
		t.Fatal(failure)
	}
	if out.Tab.ID != "tab-2" || out.Tab.IsActive {
		t.Errorf("answered %+v, want tab-2 with isActive false", out.Tab)
	}
}
