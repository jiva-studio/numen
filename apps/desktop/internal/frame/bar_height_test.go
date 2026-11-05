package frame

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
)

// frameStyles is the token file of the sizes a window frame reserves.
const frameStyles = "../../../../libs/ui/shell/src/features/window-frame/tokens/frame.css"

var barHeightValue = regexp.MustCompile(`(?m)^\s*--numen-title-bar-height:\s*(\d+)px;`)

// readDrawnBarHeight returns the height of a title bar in the page, in pixels.
func readDrawnBarHeight(t *testing.T) int {
	t.Helper()
	css, err := os.ReadFile(filepath.FromSlash(frameStyles))
	if err != nil {
		t.Fatalf("reading the styles of the frame: %v", err)
	}
	match := barHeightValue.FindSubmatch(css)
	if match == nil {
		t.Fatal("the styles of the frame declare no --numen-title-bar-height in px")
	}
	height, err := strconv.Atoi(string(match[1]))
	if err != nil {
		t.Fatal(err)
	}
	return height
}

func TestBarHeightIsTheHeightThePageDraws(t *testing.T) {
	if height := readDrawnBarHeight(t); BarHeight != height {
		t.Errorf("BarHeight = %d, the page draws a bar %d high", BarHeight, height)
	}
}
