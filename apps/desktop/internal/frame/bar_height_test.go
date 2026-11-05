package frame

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// barStyles is the stylesheet of the bar the page draws.
const barStyles = "../../../../libs/ui/shell/src/features/title-bar/ui/titleBar.css"

var (
	barRule     = regexp.MustCompile(`(?s)\.title-bar\s*\{(.*?)\}`)
	heightValue = regexp.MustCompile(`(?m)^\s*height:\s*([^;]+);`)
	pixelValue  = regexp.MustCompile(`^(\d+)px$`)
)

// readDrawnBarHeight returns the declared height of the bar in the page, as written.
func readDrawnBarHeight(t *testing.T) string {
	t.Helper()
	css, err := os.ReadFile(filepath.FromSlash(barStyles))
	if err != nil {
		t.Fatalf("reading the styles of the bar: %v", err)
	}
	rule := barRule.FindSubmatch(css)
	if rule == nil {
		t.Fatal("the styles have no .title-bar rule")
	}
	height := heightValue.FindSubmatch(rule[1])
	if height == nil {
		t.Fatal("the .title-bar rule has no height")
	}
	return string(height[1])
}

func TestBarHeightInTheStylesIsALengthInPixels(t *testing.T) {
	height := readDrawnBarHeight(t)

	if !pixelValue.MatchString(height) {
		t.Errorf("the bar is drawn %s high, want a length in px that no scale changes", height)
	}
}

func TestMacWindowIsGivenTheHeightOfTheBarThePageDraws(t *testing.T) {
	match := pixelValue.FindStringSubmatch(readDrawnBarHeight(t))
	if match == nil {
		t.Fatalf("the bar is not drawn in px: %s", readDrawnBarHeight(t))
	}
	drawn, err := strconv.Atoi(match[1])
	if err != nil {
		t.Fatal(err)
	}

	got := Apply(application.WebviewWindowOptions{}, "darwin")

	if got.Mac.InvisibleTitleBarHeight != drawn {
		t.Errorf("invisible title bar height = %d, the page draws a bar %d high", got.Mac.InvisibleTitleBarHeight, drawn)
	}
	if BarHeight != drawn {
		t.Errorf("BarHeight = %d, the page draws a bar %d high", BarHeight, drawn)
	}
}
