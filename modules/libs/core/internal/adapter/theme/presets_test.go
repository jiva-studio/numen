package theme_test

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/theme"
)

// contract defines the design token stylesheet required for theme presets.
const contract = "../../../../../libs/ui/src/shared/tokens/tokens.css"

// valueKind is the shape a value has. Each is read off the contract, so a token
// given a different shape there holds every preset to the new one.
type valueKind int

const (
	colour valueKind = iota
	length
	shadow
)

func (t valueKind) String() string { return [...]string{"a colour", "a length", "a shadow"}[t] }

type declaration struct {
	property string
	value    string
}

// A preset sets tokens, and says which half the palette is published in. There
// is nothing else in one.
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

// A token takes a colour, a length, or a shadow, which is lengths and a colour
// in one value. A preset that sets a token writes the shape that token takes.
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

// readPresets is every preset's text, under the name it is asked for.
func readPresets(t *testing.T) map[string]string {
	t.Helper()
	catalogue := createThemeFolder(t)
	texts := map[string]string{}
	for _, one := range catalogue.Themes() {
		if one.Origin != theme.Preset {
			continue
		}
		text, err := catalogue.Text(one.Name)
		if err != nil {
			t.Fatal(err)
		}
		texts[one.Name] = text
	}
	if len(texts) == 0 {
		t.Fatal("this build ships no preset")
	}
	return texts
}

// getThemeFields is what a theme sets, and the shape each of them takes.
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

// resolveToken follows a token given as another token, to the value at the end of
// it. A cycle of tokens resolves to itself and retains its raw value.
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

// getValueKind reads a value as the shape it is. A colour and a length say what they
// are, and a shadow is what is left.
func getValueKind(value string) valueKind {
	switch {
	case isColour(value):
		return colour
	case isLength(value):
		return length
	default:
		return shadow
	}
}

func hasValueKind(value string, want valueKind) bool {
	switch want {
	case colour:
		return isColour(value)
	case length:
		return isLength(value)
	default:
		return isShadow(value)
	}
}

var hex = regexp.MustCompile(`^#([0-9a-fA-F]{3,4}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})$`)

// isColour is a hex colour, or the pair of them the mode chooses between.
func isColour(value string) bool {
	if inner, is := getLightDarkPair(value); is {
		halves := splitOn(inner, ',')
		return len(halves) == 2 &&
			isColour(strings.TrimSpace(halves[0])) && isColour(strings.TrimSpace(halves[1]))
	}
	return hex.MatchString(value)
}

var measure = regexp.MustCompile(`^-?(\d+\.?\d*|\.\d+)(px|rem|em|%)$`)

func isLength(value string) bool { return value == "0" || measure.MatchString(value) }

// isShadow is one layer or several, each of them the offsets and the spread a
// shadow is cast at, with the colour it is cast in after them.
func isShadow(value string) bool {
	for _, layer := range splitOn(value, ',') {
		parts := splitWords(layer)
		if len(parts) < 3 || !isColour(parts[len(parts)-1]) {
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

// getLightDarkPair is the inside of a `light-dark()`.
func getLightDarkPair(value string) (string, bool) {
	const opens = "light-dark("
	if !strings.HasPrefix(value, opens) || !strings.HasSuffix(value, ")") {
		return "", false
	}
	return value[len(opens) : len(value)-1], true
}

// setBy is every property a stylesheet sets, whichever of its rules sets it.
func setBy(css string) []declaration {
	var found []declaration
	for _, block := range splitBlocks(stripComments(css)) {
		found = append(found, getDeclarations(block)...)
	}
	return found
}

// getDeclarations is what one block sets. A value runs to the semicolon standing
// outside brackets, so a pair written over two lines stays whole.
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

// splitBlocks is the inside of every rule in a stylesheet.
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

// stripComments is a stylesheet with its comments cut away.
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

// splitOn splits on the separators standing outside brackets.
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

// splitWords splits on the spaces standing outside brackets, so a pair is one word
// however it is laid out.
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
