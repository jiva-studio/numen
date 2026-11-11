package windowsession

import (
	"errors"
	windowdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

func reportTabs(active string) windowdomain.OpenTabs {
	return windowdomain.OpenTabs{
		ActiveID: active,
		Tabs:     []windowdomain.Tab{{ID: "note:one", Kind: "note"}, {ID: "plex:two", Kind: "plex"}},
	}
}

func TestShowingATabThatIsNotOpenFails(t *testing.T) {
	w := newTestWindow(t, nil)
	w.tabs.ReportOpenTabs(reportTabs("note:one"))

	_, err := w.scenarios.ShowTab.Execute(t.Context(), "note:missing")

	if !errors.Is(err, plugin.ErrTabNotOpen) {
		t.Errorf("showing a tab that is not open returned %v, want %v", err, plugin.ErrTabNotOpen)
	}
}

func TestShowingTheActiveTabAsksForItAndAnswersAtOnce(t *testing.T) {
	w := newTestWindow(t, nil)
	w.tabs.ReportOpenTabs(reportTabs("note:one"))

	open, err := w.scenarios.ShowTab.Execute(t.Context(), "note:one")
	if err != nil {
		t.Fatal(err)
	}

	if open.ActiveID != "note:one" || !slices.Equal(w.front.ids, []string{"note:one"}) {
		t.Errorf("the active tab is %q and the clients were asked for %v", open.ActiveID, w.front.ids)
	}
}

func TestShowingATabWaitsForTheWindowToReportItActive(t *testing.T) {
	w := newTestWindow(t, nil)
	w.tabs.ReportOpenTabs(reportTabs("note:one"))
	go func() {
		time.Sleep(20 * time.Millisecond)
		w.tabs.ReportOpenTabs(reportTabs("plex:two"))
	}()

	open, err := w.scenarios.ShowTab.Execute(t.Context(), "plex:two")
	if err != nil {
		t.Fatal(err)
	}

	if open.ActiveID != "plex:two" {
		t.Errorf("the active tab is %q, want plex:two", open.ActiveID)
	}
}

func TestATabTheWindowDoesNotShowInTimeFails(t *testing.T) {
	w := newTestWindow(t, func(deps *Deps) { deps.TabBound = 20 * time.Millisecond })
	w.tabs.ReportOpenTabs(reportTabs("note:one"))

	_, err := w.scenarios.ShowTab.Execute(t.Context(), "plex:two")

	if !errors.Is(err, plugin.ErrTabNotShown) {
		t.Errorf("showing a tab the window never activates returned %v, want %v", err, plugin.ErrTabNotShown)
	}
}
