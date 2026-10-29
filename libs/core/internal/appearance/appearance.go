// Package appearance is the settings' appearance section as embedded in a page.
//
// Every window of the installation shows the same appearance: the mode, the
// theme's own file, and the interface and text scales. The user sets these once
// and every application shows them, so how they are written into a page is
// defined here once, not per window.
package appearance

import (
	"bytes"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// HeadEnd is where the style elements are inserted: after everything the build
// put in the head, so the user's choice takes precedence.
const HeadEnd = "</head>"

// Marker is the attribute every one of those elements carries, and IsMode,
// IsTheme and IsSizes are its values. A window locates the elements by it.
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

// Settings is the user's appearance configuration.
type Settings struct {
	Mode ColorScheme
	// Theme is the stylesheet chosen, as its text. A window with none uses
	// `tokens.css`.
	Theme string
	// InterfaceScale is the scale of the interface and TextScale the scale of
	// the reading text, with 1 being the designed size.
	InterfaceScale float64
	TextScale      float64
}

// Styles is the elements the head ends with.
//
// The page contains all of them on arrival, so no frame renders with the
// default colours or an unset scale. A scale applied after the first frame
// reflows the document.
func (s Settings) Styles() string {
	// The mode first and the theme second. A theme pinning `color-scheme` is
	// the later of two declarations weighing the same, and light and dark are
	// then that theme's own.
	out := buildStyleElement(IsMode, ":root { color-scheme: "+scheme(s.Mode)+"; }")
	if s.Theme != "" {
		out += buildStyleElement(IsTheme, s.Theme)
	}
	// The two scales last. They are the user's scales for this window, within the
	// bounds of each, and the element carrying them takes precedence.
	return out + buildSizeElement(s.InterfaceScale, s.TextScale)
}

// AddStyles is the page carrying those elements, put where the head ends. A
// page with no head is handed back as it was built.
func AddStyles(text []byte, styles string) []byte {
	pos := bytes.LastIndex(text, []byte(HeadEnd))
	if styles == "" || pos < 0 {
		return text
	}
	return slices.Concat(text[:pos], []byte(styles), text[pos:])
}

// buildSizeElement is the two multipliers as the page carries them: how large the
// interface is drawn, which is the root's font size, and how large the text a
// person reads is set.
func buildSizeElement(rendered, set float64) string {
	var vars []string
	if rendered > 0 {
		vars = append(vars, "--numen-interface-scale: "+number(rendered))
	}
	if set > 0 {
		vars = append(vars, "--numen-text-scale: "+number(set))
	}
	if len(vars) == 0 {
		return ""
	}
	return buildStyleElement(IsSizes, ":root { "+strings.Join(vars, "; ")+"; }")
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

// buildStyleElement is one stylesheet as the page carries it, marked as the one of the
// three it is. A `</style>` in a theme's file is written as the CSS escape for
// it, which is the same declaration and ends no element.
func buildStyleElement(is, css string) string {
	escaped := ending.ReplaceAllStringFunc(css, func(found string) string {
		return `<\` + found[len("<"):]
	})
	return `<style ` + Marker + `="` + is + `">` + escaped + "</style>\n"
}

var ending = regexp.MustCompile(`(?i)</style`)
