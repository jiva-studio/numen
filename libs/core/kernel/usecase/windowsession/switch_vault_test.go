package windowsession

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
)

type recordedPath struct{ paths []string }

func (r *recordedPath) SetVaultPath(path string) { r.paths = append(r.paths, path) }

type countedView struct{ openedCount int }

func (c *countedView) ShowOpenTabs() { c.openedCount++ }

func TestSwitchVaultPointsThePluginsAtTheVaultOnlyWhereItWasShown(t *testing.T) {
	w := newTestWindow(t, nil)
	w.open(t, garden())
	plugins, tabs := &recordedPath{}, &countedView{}
	switching := NewSwitchVault(w.scenarios.Switching.Show, plugins, tabs)

	shownErr := switching.Execute(t.Context(), studio(), nil)
	w.pages.isAsking = true
	refusedErr := switching.Execute(t.Context(), garden(), nil)

	if shownErr != nil || !errors.Is(refusedErr, window.ErrAnswerPending) {
		t.Fatalf("errors %v and %v", shownErr, refusedErr)
	}
	if len(plugins.paths) != 1 || plugins.paths[0] != studio().Path || tabs.openedCount != 2 {
		t.Errorf("followed %v, tabs openedCount %d", plugins.paths, tabs.openedCount)
	}
}
