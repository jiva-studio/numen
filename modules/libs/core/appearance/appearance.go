// Package appearance is the settings' appearance section as a page carries it.
//
// Every window of the installation shows the same one: the mode, the theme's
// own file, and the two sizes it is drawn and set at. A person sets those once
// and every application they open shows them, so how they are written into a
// page is one thing here and not one per window.
package appearance

import (
	"bytes"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// HeadEnd is where the style elements go: after everything the build put in the
// head, so what a person chose is the last word on it.
const HeadEnd = "</head>"

// Marker is the attribute every one of those elements carries, and Mode, Theme
// and Sizes are what each says it is. A window reads them by it.
const (
	Marker  = "data-appearance"
	IsMode  = "mode"
	IsTheme = "theme"
	IsSizes = "sizes"
)

// ColorScheme is which half of every colour pair the tokens are read as.
type ColorScheme int

const (
	// System is both halves, and the machine decides between them.
	System ColorScheme = iota
	Light
	Dark
)

// Settings is what a person set their windows to.
type Settings struct {
	Mode ColorScheme
	// Theme is the stylesheet chosen, as its text. A window wearing none is
	// left to `tokens.css`.
	Theme string
	// InterfaceScale is how large the interface is drawn and TextScale how
	// large the text a person reads is set, one being as designed. A size
	// nobody named is zero and is left to `tokens.css` as well.
	InterfaceScale float64
	TextScale      float64
}

// Styles is the elements the head ends with.
//
// The page arrives carrying all of them, so no frame is drawn in the default
// colours or at a size nobody asked for. A size arriving after the first frame
// relays out the document.
func Styles(c Settings) string {
	// The mode first and the theme second. A theme pinning `color-scheme` is
	// the later of two declarations weighing the same, and light and dark are
	// then that theme's own.
	out := styled(IsMode, ":root { color-scheme: "+scheme(c.Mode)+"; }")
	if c.Theme != "" {
		out += styled(IsTheme, c.Theme)
	}
	// The two sizes last. They are what a person set this window to, inside the
	// bounds each goes to, and the element carrying them is the last word on
	// them.
	return out + sized(c.InterfaceScale, c.TextScale)
}

// Into is the page carrying those elements, put where the head ends. A page
// with no head is handed back as it was built.
func Into(text []byte, styles string) []byte {
	at := bytes.LastIndex(text, []byte(HeadEnd))
	if styles == "" || at < 0 {
		return text
	}
	return slices.Concat(text[:at], []byte(styles), text[at:])
}

// sized is the two multipliers as the page carries them: how large the
// interface is drawn, which is the root's font size, and how large the text a
// person reads is set.
func sized(drawn, set float64) string {
	var held []string
	if drawn > 0 {
		held = append(held, "--numen-interface-scale: "+number(drawn))
	}
	if set > 0 {
		held = append(held, "--numen-text-scale: "+number(set))
	}
	if len(held) == 0 {
		return ""
	}
	return styled(IsSizes, ":root { "+strings.Join(held, "; ")+"; }")
}

// number is a multiplier as CSS takes it, at the shortest that reads back as
// the number it was given.
func number(size float64) string { return strconv.FormatFloat(size, 'f', -1, 64) }

// scheme is which half of every colour pair the tokens are read as.
func scheme(mode ColorScheme) string {
	switch mode {
	case Light:
		return "light"
	case Dark:
		return "dark"
	default:
		return "light dark"
	}
}

// styled is one stylesheet as the page carries it, marked as the one of the
// three it is. A `</style>` in a theme's file is written as the CSS escape for
// it, which is the same declaration and ends no element.
func styled(is, css string) string {
	escaped := ending.ReplaceAllStringFunc(css, func(found string) string {
		return `<\` + found[len("<"):]
	})
	return `<style ` + Marker + `="` + is + `">` + escaped + "</style>\n"
}

var ending = regexp.MustCompile(`(?i)</style`)
