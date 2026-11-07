package frame

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
)

// The height of a strip of tabs at the top of a window without a bar is the token of the shell; the
// bar of the Title bar plugin declares its own.
const (
	shellTokens = "../../../../libs/ui/shell/src/features/window-frame/tokens/frame.css"
	pluginBar   = "../../../../plugins/views-title-bar/frontend/ui/titleBar.css"
)

var (
	tokenHeight = regexp.MustCompile(`(?m)^\s*--numen-title-bar-height:\s*(\d+)px;`)
	barRule     = regexp.MustCompile(`(?s)\.title-bar\s*\{(.*?)\}`)
	ruleHeight  = regexp.MustCompile(`(?m)^\s*height:\s*(\d+)px;`)
)

// readPixels returns the pixels the pattern finds in the file, failing where it finds none.
func readPixels(t *testing.T, file string, pattern *regexp.Regexp) int {
	t.Helper()
	css, err := os.ReadFile(filepath.FromSlash(file))
	if err != nil {
		t.Fatalf("reading %s: %v", file, err)
	}
	match := pattern.FindSubmatch(css)
	if match == nil {
		t.Fatalf("%s declares no height in px", file)
	}
	height, err := strconv.Atoi(string(match[1]))
	if err != nil {
		t.Fatal(err)
	}
	return height
}

func TestBarHeightIsTheHeightOfTheStripsAtTheTop(t *testing.T) {
	if height := readPixels(t, shellTokens, tokenHeight); BarHeight != height {
		t.Errorf("BarHeight = %d, the strips at the top are %d high", BarHeight, height)
	}
}

func TestBarHeightIsTheHeightOfTheBarOfThePlugin(t *testing.T) {
	css, err := os.ReadFile(filepath.FromSlash(pluginBar))
	if err != nil {
		t.Fatal(err)
	}
	rule := barRule.FindSubmatch(css)
	if rule == nil {
		t.Fatal("the styles of the bar have no .title-bar rule")
	}
	match := ruleHeight.FindSubmatch(rule[1])
	if match == nil {
		t.Fatal("the .title-bar rule has no height in px")
	}
	height, err := strconv.Atoi(string(match[1]))
	if err != nil {
		t.Fatal(err)
	}
	if BarHeight != height {
		t.Errorf("BarHeight = %d, the plugin draws a bar %d high", BarHeight, height)
	}
}
