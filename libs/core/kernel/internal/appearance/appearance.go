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

	themedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/theme"
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

// Settings is the user's appearance configuration.
type Settings struct {
	Mode           themedomain.ColorScheme
	Theme          string
	InterfaceScale float64
	TextScale      float64
}

// Styles is the elements the head ends with.
//
// The page contains all of them on arrival, so no frame renders with the
// default colors or an unset scale. A scale applied after the first frame
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
// page with no head is returned as it was built.
func AddStyles(text []byte, styles string) []byte {
	pos := bytes.LastIndex(text, []byte(HeadEnd))
	if styles == "" || pos < 0 {
		return text
	}
	return slices.Concat(text[:pos], []byte(styles), text[pos:])
}

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

func number(size float64) string { return strconv.FormatFloat(size, 'f', -1, 64) }

func scheme(mode themedomain.ColorScheme) string {
	switch mode {
	case themedomain.Light:
		return "light"
	case themedomain.Dark:
		return "dark"
	default:
		return "light dark"
	}
}

func buildStyleElement(is, css string) string {
	escaped := ending.ReplaceAllStringFunc(css, func(found string) string {
		return `<\` + found[len("<"):]
	})
	return `<style ` + Marker + `="` + is + `">` + escaped + "</style>\n"
}

var ending = regexp.MustCompile(`(?i)</style`)
