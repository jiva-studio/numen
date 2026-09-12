package theme_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/theme"
)

// waited is how long a test gives the operating system to report a change. It
// is a bound on a machine under load and not a measurement.
const waited = 5 * time.Second

// readNames is the names reported next, debounced into one batch by the watcher.
func readNames(t *testing.T, changed <-chan []string) []string {
	t.Helper()
	select {
	case names, open := <-changed:
		if !open {
			t.Fatal("the watch stopped")
		}
		return names
	case <-time.After(waited):
		t.Fatal("nothing was reported")
		return nil
	}
}

func startWatch(t *testing.T, catalogue theme.Catalogue, hold time.Duration) <-chan []string {
	t.Helper()
	changed, err := catalogue.Watch(t.Context(), hold)
	if err != nil {
		t.Fatal(err)
	}
	// The watch is established after the call returns on some systems, so the
	// first write is made once it is standing.
	time.Sleep(50 * time.Millisecond)
	return changed
}

func TestAThemeWrittenIntoTheFolderIsReported(t *testing.T) {
	catalogue := folder(t)
	changed := startWatch(t, catalogue, 20*time.Millisecond)

	put(t, catalogue, "dracula.css", ":root { --numen-surface: #282a36 }")
	if names := readNames(t, changed); len(names) != 1 || names[0] != "mine:dracula" {
		t.Errorf("reported %v", names)
	}
}

// A themes folder reached through a link is watched like any other: the system
// names a changed file by the path the link leads to.
func TestAThemeFolderReachedThroughALinkIsWatched(t *testing.T) {
	real := filepath.Join(t.TempDir(), "elsewhere")
	if err := os.MkdirAll(real, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "themes")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("this machine does not make links: %v", err)
	}

	catalogue, err := theme.At(link)
	if err != nil {
		t.Fatal(err)
	}
	changed := startWatch(t, catalogue, 20*time.Millisecond)

	body := ":root { --numen-surface: #1c1c28 }"
	if err := os.WriteFile(filepath.Join(real, "midnight.css"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if names := readNames(t, changed); len(names) != 1 || names[0] != "mine:midnight" {
		t.Errorf("reported %v", names)
	}
}

// An editor saves through a temporary file and a rename, so the watch is on the
// folder and the theme is named however the bytes arrive.
func TestAThemeSavedOverItselfIsReported(t *testing.T) {
	catalogue := folder(t)
	put(t, catalogue, "dracula.css", ":root { --numen-surface: #282a36 }")
	changed := startWatch(t, catalogue, 20*time.Millisecond)

	tmp := filepath.Join(catalogue.Dir(), ".dracula.css.swp")
	if err := os.WriteFile(tmp, []byte(":root { --numen-surface: #000000 }"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(tmp, filepath.Join(catalogue.Dir(), "dracula.css")); err != nil {
		t.Fatal(err)
	}
	if names := readNames(t, changed); len(names) != 1 || names[0] != "mine:dracula" {
		t.Errorf("reported %v", names)
	}
}

// What changed about a theme that is gone is that it is gone, and it is named
// here too.
func TestAThemeDeletedIsReportedByName(t *testing.T) {
	catalogue := folder(t)
	put(t, catalogue, "dracula.css", ":root {}")
	changed := startWatch(t, catalogue, 20*time.Millisecond)

	if err := os.Remove(filepath.Join(catalogue.Dir(), "dracula.css")); err != nil {
		t.Fatal(err)
	}
	if names := readNames(t, changed); len(names) != 1 || names[0] != "mine:dracula" {
		t.Errorf("reported %v", names)
	}
}

// One file that is not a theme, and one that is: what is reported is the theme
// alone.
func TestWhatIsNotAThemeIsNotReported(t *testing.T) {
	catalogue := folder(t)
	changed := startWatch(t, catalogue, 20*time.Millisecond)

	put(t, catalogue, "dracula.css.bak", ":root {}")
	put(t, catalogue, "notes.md", "the themes I mean to write")
	if err := os.MkdirAll(filepath.Join(catalogue.Dir(), "kept"), 0o755); err != nil {
		t.Fatal(err)
	}
	put(t, catalogue, "nord.css", ":root {}")

	if names := readNames(t, changed); len(names) != 1 || names[0] != "mine:nord" {
		t.Errorf("reported %v", names)
	}
}

// Several files landing at once are one report: a folder copied in is one
// thing a person did.
func TestAHandfulOfFilesLandingAtOnceIsOneReport(t *testing.T) {
	catalogue := folder(t)
	changed := startWatch(t, catalogue, 300*time.Millisecond)

	for _, name := range []string{"nord.css", "dracula.css", "gruvbox.css"} {
		put(t, catalogue, name, ":root {}")
	}
	names := readNames(t, changed)
	if len(names) != 3 {
		t.Errorf("reported %v", names)
	}
}

func TestAWatchOnAFolderThatIsNotThereIsRefused(t *testing.T) {
	var catalogue theme.Catalogue
	if _, err := catalogue.Watch(t.Context(), 0); err == nil {
		t.Error("a catalogue with no folder was watched")
	}
}

// The watch ends where the caller does, and the channel is closed behind it.
func TestAWatchStopsWithTheCallerItWasStartedFor(t *testing.T) {
	ctx, stop := context.WithCancel(t.Context())
	changed, err := folder(t).Watch(ctx, 20*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	stop()

	for {
		select {
		case _, open := <-changed:
			if !open {
				return
			}
		case <-time.After(waited):
			t.Fatal("the watch went on after the caller left")
		}
	}
}
