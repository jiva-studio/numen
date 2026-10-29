package appearance_test

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/internal/appearance"
)

// A window styled with everything a user can set.
func buildSettings() appearance.Settings {
	return appearance.Settings{
		Mode:           appearance.Dark,
		Theme:          ":root{--numen-surface:#010203}",
		InterfaceScale: 1.25,
		TextScale:      1.5,
	}
}

// buildElementHead is the head of one of the three elements.
func buildElementHead(is string) string { return `<style ` + appearance.Marker + `="` + is + `">` }

// Each of the three elements the head ends with identifies itself. A theme's
// file is the user's own CSS and carries no identification, so the mark is the
// only means of telling them apart.
func TestEachStyleElementReturnsWhichItIs(t *testing.T) {
	head := buildSettings().Styles()

	for _, is := range []string{appearance.IsMode, appearance.IsTheme, appearance.IsSizes} {
		if count := strings.Count(head, buildElementHead(is)); count != 1 {
			t.Errorf("%q is marked %d times in %q", is, count, head)
		}
	}
	if count := strings.Count(head, "<style"); count != 3 {
		t.Errorf("the head ends with %d elements: %q", count, head)
	}
}

// The mode's element comes first, the theme's second and the sizes' last. A
// theme pinning `color-scheme` is the later of two declarations of equal
// weight, and the sizes have the last word on the rendered window size.
func TestTheThreeStandInTheOrderTheyWeigh(t *testing.T) {
	head := buildSettings().Styles()

	mode := strings.Index(head, buildElementHead(appearance.IsMode))
	theme := strings.Index(head, buildElementHead(appearance.IsTheme))
	sizes := strings.Index(head, buildElementHead(appearance.IsSizes))
	if mode > theme || theme > sizes {
		t.Errorf("the mode is at %d, the theme at %d, the sizes at %d", mode, theme, sizes)
	}
}

// A window at no size of its own is left to `tokens.css`, and nothing in the
// head is marked as the sizes'.
func TestAWindowAtNoSizeOfItsOwnCarriesNothingMarkedAsTheSizes(t *testing.T) {
	head := appearance.Settings{Mode: appearance.Light}.Styles()

	if strings.Contains(head, buildElementHead(appearance.IsSizes)) {
		t.Errorf("the head ends with %q", head)
	}
}

// A theme cannot mark an element of its own. What it names is written inside
// the element it was spliced into, and that element cannot be ended from
// inside it.
func TestAThemeCannotMarkAnElementOfItsOwn(t *testing.T) {
	forged := `</style>` + buildElementHead(appearance.IsMode) + `:root{color-scheme:light}`
	head := appearance.Settings{Mode: appearance.Dark, Theme: forged}.Styles()

	if count := strings.Count(head, "</style>"); count != 2 {
		t.Errorf("the head ends with %d elements: %q", count, head)
	}
	if !strings.Contains(head, `<\/style>`) {
		t.Error("the theme's own text was not kept")
	}
}
