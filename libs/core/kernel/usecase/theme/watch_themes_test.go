package theme_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	themedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/theme"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/themefiles"
	themeusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/theme"
)

// reported collects the names a watch sends until every wanted name has come, and stops the watch.
type reported struct {
	names   chan []string
	stopped chan error
}

func startWatch(t *testing.T, desk *themesDesk) *reported {
	t.Helper()
	watching := &reported{names: make(chan []string, 64), stopped: make(chan error, 1)}
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	go func() {
		watching.stopped <- themeusecase.NewWatchThemes(desk.sources, desk.files, 20*time.Millisecond).
			Execute(ctx, func(names []string) error {
				watching.names <- names
				return nil
			})
	}()
	// The watch is established after the call returns on some systems, so the first change is made
	// once it is standing.
	time.Sleep(100 * time.Millisecond)
	return watching
}

func (r *reported) await(t *testing.T, want ...string) {
	t.Helper()
	reports := map[string]bool{}
	deadline := time.After(10 * time.Second)
	for {
		select {
		case names := <-r.names:
			for _, name := range names {
				reports[name] = true
			}
			if !slices.ContainsFunc(want, func(name string) bool { return !reports[name] }) {
				return
			}
		case err := <-r.stopped:
			t.Fatalf("the watch ended with %v reported, want %v: %v", reports, want, err)
		case <-deadline:
			t.Fatalf("reported %v, want %v", reports, want)
		}
	}
}

func TestAThemeSavedIntoTheFolderIsReportedByName(t *testing.T) {
	desk := createThemesDesk(t, themedomain.Appearance{})
	watching := startWatch(t, desk)

	desk.writeTheme(t, "dracula.css", ":root { --numen-surface: #282a36 }")

	watching.await(t, "mine:dracula")
}

func TestWatchThemesReportsThePluginThemesOfAPluginDisabled(t *testing.T) {
	desk := createThemesDesk(t, themedomain.Appearance{})
	watching := startWatch(t, desk)

	desk.plugins.disable()

	watching.await(t, themedomain.PluginName(pluginID, "night"), themedomain.PluginName(pluginID, "day"))
}

func TestWatchThemesReportsThePluginThemesOfAPluginEnabled(t *testing.T) {
	desk := createThemesDesk(t, themedomain.Appearance{})
	desk.plugins.disable()
	watching := startWatch(t, desk)

	desk.plugins.enable(createFakePlugins().themes...)

	watching.await(t, themedomain.PluginName(pluginID, "night"), themedomain.PluginName(pluginID, "day"))
}

// A build with no folder but with plugins still streams.
func TestWatchThemesOfABuildWithoutAFolderStillWatchesPlugins(t *testing.T) {
	desk := createThemesDesk(t, themedomain.Appearance{})
	var withoutFolder themefiles.Files
	desk.sources.Own = withoutFolder
	desk.files = withoutFolder
	watching := startWatch(t, desk)

	desk.plugins.disable()

	watching.await(t, themedomain.PluginName(pluginID, "night"))
}

func TestWatchThemesOfABuildWithoutAFolderOrPluginsIsRefused(t *testing.T) {
	desk := createThemesDesk(t, themedomain.Appearance{})
	desk.sources.Plugins = nil
	err := themeusecase.NewWatchThemes(desk.sources, themefiles.Files{}, 0).
		Execute(t.Context(), func([]string) error { return nil })
	if !errors.Is(err, themeusecase.ErrWatchUnavailable) {
		t.Errorf("error = %v, want ErrWatchUnavailable", err)
	}
}
