package frame

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
)

// shellTokens holds the height of a strip of tabs at the top of a window, a token of the shell.
const shellTokens = "../../../../libs/ui/shell/src/features/window-frame/tokens/frame.css"

var tokenHeight = regexp.MustCompile(`(?m)^\s*--numen-title-bar-height:\s*(\d+)px;`)

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
