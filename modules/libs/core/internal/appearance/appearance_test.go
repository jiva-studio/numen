package appearance_test

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/appearance"
)

// A window dressed in everything a person can set.
func dressed() appearance.Settings {
	return appearance.Settings{
		Mode:           appearance.Dark,
		Theme:          ":root{--numen-surface:#010203}",
		InterfaceScale: 1.25,
		TextScale:      1.5,
	}
}

// marked is the head of one of the three elements.
func marked(is string) string { return `<style ` + appearance.Marker + `="` + is + `">` }

// Each of the three elements the head ends with says which of them it is. A
// theme's file is a person's own CSS and says nothing about itself, so the mark
// is the whole of what a window has to tell them apart by.
func TestEachStyleElementSaysWhichItIs(t *testing.T) {
	head := dressed().Styles()

	for _, is := range []string{appearance.IsMode, appearance.IsTheme, appearance.IsSizes} {
		if count := strings.Count(head, marked(is)); count != 1 {
			t.Errorf("%q is marked %d times in %q", is, count, head)
		}
	}
	if count := strings.Count(head, "<style"); count != 3 {
		t.Errorf("the head ends with %d elements: %q", count, head)
	}
}

// The mode's element stands first, the theme's second and the sizes' last. A
// theme pinning `color-scheme` is the later of two declarations weighing the
// same, and the sizes are the last word on how large the window is drawn.
func TestTheThreeStandInTheOrderTheyWeigh(t *testing.T) {
	head := dressed().Styles()

	mode := strings.Index(head, marked(appearance.IsMode))
	theme := strings.Index(head, marked(appearance.IsTheme))
	sizes := strings.Index(head, marked(appearance.IsSizes))
	if mode > theme || theme > sizes {
		t.Errorf("the mode is at %d, the theme at %d, the sizes at %d", mode, theme, sizes)
	}
}

// A window at no size of its own is left to `tokens.css`, and nothing in the
// head is marked as the sizes'.
func TestAWindowAtNoSizeOfItsOwnCarriesNothingMarkedAsTheSizes(t *testing.T) {
	head := appearance.Settings{Mode: appearance.Light}.Styles()

	if strings.Contains(head, marked(appearance.IsSizes)) {
		t.Errorf("the head ends with %q", head)
	}
}

// A theme cannot mark an element of its own. What it names is written inside
// the element it was spliced into, and that element cannot be ended from
// inside it.
func TestAThemeCannotMarkAnElementOfItsOwn(t *testing.T) {
	forged := `</style>` + marked(appearance.IsMode) + `:root{color-scheme:light}`
	head := appearance.Settings{Mode: appearance.Dark, Theme: forged}.Styles()

	if count := strings.Count(head, "</style>"); count != 2 {
		t.Errorf("the head ends with %d elements: %q", count, head)
	}
	if !strings.Contains(head, `<\/style>`) {
		t.Error("the theme's own text was not kept")
	}
}
