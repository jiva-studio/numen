package app

import (
	"runtime"
	"strings"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
	windowdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/window"
)

func hasSystemFrame() bool { return runtime.GOOS == "darwin" }

func TestMainWindowOptions_DropsSystemFrame(t *testing.T) {
	opened, _ := windowOn(t)

	got := getMainWindowOptions(opened)

	if got.Frameless == hasSystemFrame() {
		t.Errorf("frameless = %v on %s", got.Frameless, runtime.GOOS)
	}
	if got.Title == "" {
		t.Error("the window has no title for the taskbar")
	}
	if got.Width != 1280 || got.Height != 860 {
		t.Errorf("size = %dx%d, want 1280x860", got.Width, got.Height)
	}
}

func TestSecondaryWindowOptions_DropsSystemFrame(t *testing.T) {
	opened, _ := windowOn(t)

	for _, c := range []struct {
		tab     windowdomain.DetachedTab
		wantPos application.WindowStartPosition
		wantX   int
		wantY   int
	}{
		{windowdomain.DetachedTab{ID: "a", X: 0, Y: 0}, application.WindowCentered, 0, 0},
		{windowdomain.DetachedTab{ID: "a", X: 1, Y: 0}, application.WindowXY, 1, 0},
		{windowdomain.DetachedTab{ID: "a", X: -1, Y: 0}, application.WindowXY, -1, 0},
		{windowdomain.DetachedTab{ID: "a", X: 0, Y: 1}, application.WindowXY, 0, 1},
		{windowdomain.DetachedTab{ID: "a", X: 0, Y: -1}, application.WindowXY, 0, -1},
		{windowdomain.DetachedTab{ID: "a", X: 10, Y: 20}, application.WindowXY, 10, 20},
	} {
		got := getSecondaryWindowOptions(opened, c.tab)
		if got.Frameless == hasSystemFrame() {
			t.Errorf("frameless = %v on %s", got.Frameless, runtime.GOOS)
		}
		if got.InitialPosition != c.wantPos || got.X != c.wantX || got.Y != c.wantY {
			t.Errorf("tab %+v: pos = %v (%d,%d), want %v (%d,%d)",
				c.tab, got.InitialPosition, got.X, got.Y, c.wantPos, c.wantX, c.wantY)
		}
	}
}

func TestSecondaryWindowOptions_UsesDetachedSize(t *testing.T) {
	opened, _ := windowOn(t)

	for _, c := range []struct {
		tab        windowdomain.DetachedTab
		wantWidth  int
		wantHeight int
	}{
		{windowdomain.DetachedTab{ID: "a", Width: 640, Height: 480}, 640, 480},
		{windowdomain.DetachedTab{ID: "a", Width: 1, Height: 1}, 1, 1},
		{windowdomain.DetachedTab{ID: "a", Width: 0, Height: 480}, 1280, 860},
		{windowdomain.DetachedTab{ID: "a", Width: 640, Height: 0}, 1280, 860},
		{windowdomain.DetachedTab{ID: "a", Width: -1, Height: 480}, 1280, 860},
		{windowdomain.DetachedTab{ID: "a", Width: 640, Height: -1}, 1280, 860},
	} {
		got := getSecondaryWindowOptions(opened, c.tab)
		if got.Width != c.wantWidth || got.Height != c.wantHeight {
			t.Errorf("tab %+v: size = %dx%d, want %dx%d",
				c.tab, got.Width, got.Height, c.wantWidth, c.wantHeight)
		}
	}
}

func TestSecondaryWindowOptions_UsesDefaultSizeWithoutOne(t *testing.T) {
	opened, _ := windowOn(t)

	got := getSecondaryWindowOptions(opened, windowdomain.DetachedTab{ID: "a"})

	if got.Width != 1280 || got.Height != 860 {
		t.Errorf("size = %dx%d, want 1280x860", got.Width, got.Height)
	}
}

func TestFatalReportPage_DrawsTitleBar(t *testing.T) {
	page, err := fatalReport{}.buildPage(container.Config{}, errFatalSample)
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{"--wails-draggable: drag", "data-close"} {
		if !strings.Contains(string(page), want) {
			t.Errorf("the report page lacks %q", want)
		}
	}
}

func TestFatalWindowOptions_DropsSystemFrame(t *testing.T) {
	got := getFatalWindowOptions()

	if got.Frameless == hasSystemFrame() {
		t.Errorf("frameless = %v on %s", got.Frameless, runtime.GOOS)
	}
	if got.Width != 720 || got.Height != 520 {
		t.Errorf("size = %dx%d, want 720x520", got.Width, got.Height)
	}
}

var errFatalSample = stringError("sample")

type stringError string

func (e stringError) Error() string { return string(e) }

func TestFatalReportPage_DrawsNoPluginAction(t *testing.T) {
	page, err := fatalReport{}.buildPage(container.Config{}, errFatalSample)
	if err != nil {
		t.Fatal(err)
	}

	for _, unwanted := range []string{"data-title-bar-command", "data-title-bar-more", "/plugin/"} {
		if strings.Contains(string(page), unwanted) {
			t.Errorf("the report page contains %q", unwanted)
		}
	}
	if !strings.Contains(string(page), "data-close") {
		t.Error("the report page lacks the close button of the bar")
	}
}
