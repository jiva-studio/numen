package settings

import (
	"errors"
	"os"
	"strconv"

	"github.com/jiva-studio/numen/modules/libs/core/appearance"
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

	// Theme is the stylesheet the window wears, named by the shelf it came off
	// and its filename: `preset:dracula` ships here, `mine:dracula` is the
	// person's file.
	Theme string `json:"theme"`

	// HangPartsUnderANode is whether a node in the plex hangs the headings of
	// its note under the box. A file leaving it out hangs them, and a file
	// naming false leaves the box alone.
	HangPartsUnderANode *bool `json:"hang_parts_under_a_node"`

	// PartsUnderANode is how many of those headings stand under a node at once,
	// the rest being wound to. A number outside PartsUnderANodeBounds is
	// refused, and a file naming none stands DefaultParts of them.
	PartsUnderANode int `json:"parts_under_a_node"`
}

// Hangs is whether a node hangs the headings of its note under it. A section
// naming nothing hangs them.
func (a Appearance) Hangs() bool {
	return a.HangPartsUnderANode == nil || *a.HangPartsUnderANode
}

// The modes a colour pair is read by.
const (
	ModeSystem = "system"
	ModeLight  = "light"
	ModeDark   = "dark"
)

// Mode is the word this file writes read back, and Word is the word for one. A
// word this file does not name is the machine's own choice.
func Mode(said string) appearance.ColorScheme {
	switch said {
	case ModeLight:
		return appearance.Light
	case ModeDark:
		return appearance.Dark
	}
	return appearance.System
}

func Word(mode appearance.ColorScheme) string {
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

// DefaultParts is how many headings stand under a node where the file names no
// number.
const DefaultParts = 6

// How far each of the two sizes goes, and how many headings a node may hang.
//
// The interface holds while the smallest control it draws is a target a
// pointer finds, and while a window 1280 across still stands its panes side by
// side. Text holds while the smallest of it is still read, and while a line of
// typing still fits the row it is typed in. A node hangs at least one heading,
// and twelve of them reach the foot of a window the plex is drawn in.
var (
	InterfaceScaleBounds  = Bounds{Least: 0.8, Most: 2}
	TextScaleBounds       = Bounds{Least: 0.8, Most: 1.75}
	PartsUnderANodeBounds = Bounds{Least: 1, Most: 12}
)

// Outsides is every number the section holds that its setting does not take,
// in the order the section names them.
func (a Appearance) Outsides() []*OutsideBounds {
	var found []*OutsideBounds
	for _, err := range []error{
		InterfaceScaleBounds.Check("appearance.interface_scale", a.InterfaceScale),
		TextScaleBounds.Check("appearance.text_scale", a.TextScale),
		PartsUnderANodeBounds.Check("appearance.parts_under_a_node", float64(a.PartsUnderANode)),
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
	if found := a.Outsides(); len(found) > 0 {
		return found[0]
	}
	return nil
}

// DefaultTheme is this product's own palette, which is what an installation
// nobody has dressed wears.
const DefaultTheme = "preset:numen"

// fromDesktop is how large the interface is drawn where the file names no size.
//
// A screen says how many pixels it has and not how large they are, so the
// desktop is asked: GDK_DPI_SCALE is what a person told their session text
// should be scaled by, and the interface is drawn to match. A number outside
// InterfaceScaleBounds is not one the setting is seeded with.
func fromDesktop() float64 {
	scale, err := strconv.ParseFloat(os.Getenv("GDK_DPI_SCALE"), 64)
	if err != nil || !InterfaceScaleBounds.Holds(scale) {
		return AsDesigned
	}
	return scale
}
