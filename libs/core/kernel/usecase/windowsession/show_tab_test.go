package windowsession

import (
	"errors"
	"io/fs"
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/note/domain"
)

func reportTabs(active string) domain.OpenTabs {
	return domain.OpenTabs{
		ActiveID: active,
		Tabs:     []domain.Tab{{ID: "note:one", Kind: "note"}, {ID: "plex:two", Kind: "plex"}},
	}
}

func TestShowingATabThatIsNotOpenFails(t *testing.T) {
	w := newTestWindow(t, nil)
	w.rt.ReportOpenTabs(reportTabs("note:one"))

	_, err := w.scenarios.ShowTab.Execute(t.Context(), "note:missing")

	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("showing a tab that is not open returned %v, want %v", err, fs.ErrNotExist)
	}
}

func TestShowingTheActiveTabAsksForItAndAnswersAtOnce(t *testing.T) {
	w := newTestWindow(t, nil)
	w.rt.ReportOpenTabs(reportTabs("note:one"))

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
	w.rt.ReportOpenTabs(reportTabs("note:one"))
	go func() {
		time.Sleep(20 * time.Millisecond)
		w.rt.ReportOpenTabs(reportTabs("plex:two"))
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
	w.rt.ReportOpenTabs(reportTabs("note:one"))

	_, err := w.scenarios.ShowTab.Execute(t.Context(), "plex:two")

	if !errors.Is(err, plugin.ErrTabNotShown) {
		t.Errorf("showing a tab the window never activates returned %v, want %v", err, plugin.ErrTabNotShown)
	}
}
