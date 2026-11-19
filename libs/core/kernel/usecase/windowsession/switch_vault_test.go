package windowsession

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
)

type recordedPath struct{ paths []string }

func (r *recordedPath) SetVaultPath(path string) { r.paths = append(r.paths, path) }

func TestSwitchVaultPointsThePluginsAtTheVaultOnlyWhereItWasShown(t *testing.T) {
	w := newTestWindow(t, nil)
	w.open(t, garden())
	plugins := &recordedPath{}
	tabs := &TabReports{}
	var shownTabs []window.OpenTabs
	tabs.ReportOpenTabs(window.OpenTabs{ActiveID: "first"})
	switching := NewSwitchVault(w.useCases.Switching.Show, plugins, tabs, func(open window.OpenTabs) { shownTabs = append(shownTabs, open) })

	shownErr := switching.Execute(t.Context(), studio(), nil)
	tabs.ReportOpenTabs(window.OpenTabs{ActiveID: "second"})
	w.pages.isAsking = true
	refusedErr := switching.Execute(t.Context(), garden(), nil)

	if shownErr != nil || !errors.Is(refusedErr, window.ErrAnswerPending) {
		t.Fatalf("errors %v and %v", shownErr, refusedErr)
	}
	if len(plugins.paths) != 1 || plugins.paths[0] != studio().Path || len(shownTabs) != 2 || shownTabs[0].ActiveID != "first" || shownTabs[1].ActiveID != "second" {
		t.Errorf("followed %v, tabs %v", plugins.paths, shownTabs)
	}
}
