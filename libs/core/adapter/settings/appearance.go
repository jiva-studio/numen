package settings

import (
	"errors"
	"os"
	"strconv"

	"github.com/jiva-studio/numen/libs/core/internal/appearance"
)

// Appearance is how the window is drawn.
type Appearance struct {
	// InterfaceScale is how large the window is drawn: its chrome, its controls,
	// the spacing between them and the type in them. A number outside
	// InterfaceScaleBounds is refused, and a file naming no size at all is drawn
	// at what the desktop asks for.
	InterfaceScale float64 `json:"interface_scale"`

	// TextScale is how large the text a person reads is set: a note, a book, an
	// answer, the editor. A number outside TextScaleBounds is refused.
	TextScale float64 `json:"text_scale"`

	// Mode is which half of a colour pair the window takes: ModeSystem,
	// ModeLight or ModeDark. A theme that pins the two halves itself leaves
	// this nothing to choose.
	Mode string `json:"mode"`

	// Theme is the stylesheet applied to the window, named by its origin
	// and its filename: `preset:dracula` ships here, `mine:dracula` is the
	// person's file.
	Theme string `json:"theme"`
}

// The modes a colour pair is read by.
const (
	ModeSystem = "system"
	ModeLight  = "light"
	ModeDark   = "dark"
)

// Mode is the word this file writes read back, and Word is the word for one. A
// word this file does not name is the machine's own choice.
func Mode(word string) appearance.ColorScheme {
	switch word {
	case ModeLight:
		return appearance.Light
	case ModeDark:
		return appearance.Dark
	}
	return appearance.System
}

func GetModeName(mode appearance.ColorScheme) string {
	switch mode {
	case appearance.Light:
		return ModeLight
	case appearance.Dark:
		return ModeDark
	default:
		return ModeSystem
	}
}

// AsDesigned is the multiplier that draws everything the size it was drawn at.
const AsDesigned = 1

// How far each of the two sizes goes.
//
// The interface holds while the smallest control it draws is a target a
// pointer finds, and while a window 1280 across still stands its panes side by
// side. Text holds while the smallest of it is still read, and while a line of
// typing still fits the row it is typed in.
var (
	InterfaceScaleBounds = Bounds{Min: 0.8, Max: 2}
	TextScaleBounds      = Bounds{Min: 0.8, Max: 1.75}
)

// GetOutsideBounds is every number the section holds that its setting does not take,
// in the order the section names them.
func (a Appearance) GetOutsideBounds() []*OutsideBounds {
	var found []*OutsideBounds
	for _, err := range []error{
		InterfaceScaleBounds.Check("appearance.interface_scale", a.InterfaceScale),
		TextScaleBounds.Check("appearance.text_scale", a.TextScale),
	} {
		var outside *OutsideBounds
		if errors.As(err, &outside) {
			found = append(found, outside)
		}
	}
	return found
}

// Check is what is wrong with the two sizes and the count, and nothing where
// each is a number its setting takes.
func (a Appearance) Check() error {
	if found := a.GetOutsideBounds(); len(found) > 0 {
		return found[0]
	}
	return nil
}

// DefaultTheme is this product's own palette, which is what an installation
// nobody has dressed wears.
const DefaultTheme = "preset:numen"

// getDesktopScale is how large the interface is drawn where the file names no size.
//
// A screen says how many pixels it has and not how large they are, so the
// desktop is asked: GDK_DPI_SCALE is what a person told their session text
// should be scaled by, and the interface is drawn to match. A number outside
// InterfaceScaleBounds is not one the setting is seeded with.
func getDesktopScale() float64 {
	scale, err := strconv.ParseFloat(os.Getenv("GDK_DPI_SCALE"), 64)
	if err != nil || !InterfaceScaleBounds.Contains(scale) {
		return AsDesigned
	}
	return scale
}
