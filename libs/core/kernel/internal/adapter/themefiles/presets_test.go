package themefiles_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
)

// contract defines the design token stylesheet required for theme presets.
const contract = "../../../../../../libs/ui/components/src/shared/tokens/tokens.css"

// valueKind is the shape a value has. Each is read off the contract, so a token
// given a different shape there holds every preset to the new one.
type valueKind int

const (
	color valueKind = iota
	length
	shadow
)

func (t valueKind) String() string { return [...]string{"a color", "a length", "a shadow"}[t] }

type declaration struct {
	property string
	value    string
}

// A theme of the presets plugin and the default theme set tokens and states which half the palette is published in. It
// contains nothing else.
func TestAPresetSetsTokensAndTheHalfItIsPublishedIn(t *testing.T) {
	sets := getThemeFields(t)
	for name, css := range readPresets(t) {
		for _, one := range setBy(css) {
			if !strings.HasPrefix(one.property, "--") {
				if one.property != "color-scheme" {
					t.Errorf("%s declares %s", name, one.property)
				} else if one.value != "light" && one.value != "dark" {
					t.Errorf("%s is published in %q", name, one.value)
				}
				continue
			}
			if _, is := sets[one.property]; !is {
				t.Errorf("%s sets %s, which is not a token a theme sets", name, one.property)
			}
		}
	}
}

// A token takes a color, a length, or an elevation, which combines lengths and
// a color in one value. A preset that sets a token uses the shape that token
// takes.
func TestAPresetProducesEachTokenTheShapeItConsumes(t *testing.T) {
	sets := getThemeFields(t)
	for name, css := range readPresets(t) {
		for _, one := range setBy(css) {
			want, is := sets[one.property]
			if !is {
				continue
			}
			if !hasValueKind(one.value, want) {
				t.Errorf("%s gives %s %q, and it takes %s", name, one.property, one.value, want)
			}
		}
	}
}

// presetsPlugin is the plugin that holds the preset themes the application does not ship.
const presetsPlugin = "../../../../../../plugins/themes-presets"

func readPresets(t *testing.T) map[string]string {
	t.Helper()
	files := createThemeFolder(t)
	texts := map[string]string{}
	for _, one := range files.ListPresets() {
		texts[one.Title] = one.Text
	}
	declared, err := manifestjson.ReadFile(filepath.Join(presetsPlugin, manifest.FileName))
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range declared.Contributes.Themes {
		text, err := manifestjson.ReadThemeFile(presetsPlugin, one.File)
		if err != nil {
			t.Fatal(err)
		}
		texts[one.ID] = text
	}
	if len(texts) <= 1 {
		t.Fatal("the presets plugin declares no theme")
	}
	return texts
}

func getThemeFields(t *testing.T) map[string]valueKind {
	t.Helper()
	css, err := os.ReadFile(contract)
	if err != nil {
		t.Fatal(err)
	}
	for _, block := range splitBlocks(stripComments(string(css))) {
		sets := map[string]valueKind{}
		target := map[string]string{}
		for _, one := range getDeclarations(block) {
			if strings.HasPrefix(one.property, "--") {
				target[one.property] = one.value
			}
		}
		for property, value := range target {
			// A token given as another token takes the shape that one takes:
			// two names for one value are one value.
			sets[property] = getValueKind(resolveToken(value, target))
		}
		if len(sets) > 0 {
			return sets
		}
	}
	t.Fatalf("%s names no token", contract)
	return nil
}

// pointing is a value that is nothing but another token: `var(--numen-hushed)`.
var pointing = regexp.MustCompile(`^var\(\s*(--[\w-]+)\s*\)$`)

func resolveToken(value string, tokens map[string]string) string {
	for range len(tokens) {
		found := pointing.FindStringSubmatch(value)
		if found == nil {
			return value
		}
		next, is := tokens[found[1]]
		if !is {
			return value
		}
		value = next
	}
	return value
}

func getValueKind(value string) valueKind {
	switch {
	case isColor(value):
		return color
	case isLength(value):
		return length
	default:
		return shadow
	}
}

func hasValueKind(value string, want valueKind) bool {
	switch want {
	case color:
		return isColor(value)
	case length:
		return isLength(value)
	default:
		return isElevationValue(value)
	}
}

var hex = regexp.MustCompile(`^#([0-9a-fA-F]{3,4}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})$`)

func isColor(value string) bool {
	if inner, is := getLightDarkPair(value); is {
		halves := splitOn(inner, ',')
		return len(halves) == 2 &&
			isColor(strings.TrimSpace(halves[0])) && isColor(strings.TrimSpace(halves[1]))
	}
	return hex.MatchString(value)
}

var measure = regexp.MustCompile(`^-?(\d+\.?\d*|\.\d+)(px|rem|em|%)$`)

func isLength(value string) bool { return value == "0" || measure.MatchString(value) }

func isElevationValue(value string) bool {
	for _, layer := range splitOn(value, ',') {
		parts := splitWords(layer)
		if len(parts) < 3 || !isColor(parts[len(parts)-1]) {
			return false
		}
		for _, one := range parts[:len(parts)-1] {
			if !isLength(one) {
				return false
			}
		}
	}
	return true
}

func getLightDarkPair(value string) (string, bool) {
	const opens = "light-dark("
	if !strings.HasPrefix(value, opens) || !strings.HasSuffix(value, ")") {
		return "", false
	}
	return value[len(opens) : len(value)-1], true
}

func setBy(css string) []declaration {
	var found []declaration
	for _, block := range splitBlocks(stripComments(css)) {
		found = append(found, getDeclarations(block)...)
	}
	return found
}

func getDeclarations(block string) []declaration {
	var found []declaration
	for _, part := range splitOn(block, ';') {
		name, value, is := strings.Cut(part, ":")
		name, value = strings.TrimSpace(name), strings.Join(splitWords(value), " ")
		if !is || name == "" || value == "" {
			continue
		}
		found = append(found, declaration{name, value})
	}
	return found
}

func splitBlocks(css string) []string {
	var found []string
	for {
		open := strings.Index(css, "{")
		if open < 0 {
			return found
		}
		shutIdx := strings.Index(css[open:], "}")
		if shutIdx < 0 {
			return found
		}
		found = append(found, css[open+1:open+shutIdx])
		css = css[open+shutIdx+1:]
	}
}

func stripComments(css string) string {
	var text strings.Builder
	for {
		start := strings.Index(css, "/*")
		if start < 0 {
			text.WriteString(css)
			return text.String()
		}
		text.WriteString(css[:start])
		end := strings.Index(css[start+2:], "*/")
		if end < 0 {
			return text.String()
		}
		css = css[start+2+end+2:]
	}
}

func splitOn(text string, on byte) []string {
	var parts []string
	depth, start := 0, 0
	for item := range len(text) {
		switch text[item] {
		case '(':
			depth++
		case ')':
			depth--
		case on:
			if depth == 0 {
				parts = append(parts, text[start:item])
				start = item + 1
			}
		}
	}
	return append(parts, text[start:])
}

func splitWords(text string) []string {
	var found []string
	depth, start := 0, 0
	keep := func(end int) {
		if word := strings.TrimSpace(text[start:end]); word != "" {
			found = append(found, word)
		}
	}
	for item := range len(text) {
		switch text[item] {
		case '(':
			depth++
		case ')':
			depth--
		case ' ', '\t', '\n', '\r':
			if depth == 0 {
				keep(item)
				start = item + 1
			}
		}
	}
	keep(len(text))
	return found
}
