package app

import (
	"runtime"
	"strings"
	"testing"

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
}

func TestSecondaryWindowOptions_DropsSystemFrame(t *testing.T) {
	opened, _ := windowOn(t)

	got := getSecondaryWindowOptions(opened, windowdomain.DetachedTab{ID: "a", X: 10, Y: 20})

	if got.Frameless == hasSystemFrame() {
		t.Errorf("frameless = %v on %s", got.Frameless, runtime.GOOS)
	}
	if got.X != 10 || got.Y != 20 {
		t.Errorf("position = %d,%d, want 10,20", got.X, got.Y)
	}
}

func TestSecondaryWindowOptions_UsesDetachedSize(t *testing.T) {
	opened, _ := windowOn(t)

	got := getSecondaryWindowOptions(opened, windowdomain.DetachedTab{ID: "a", Width: 640, Height: 480})

	if got.Width != 640 || got.Height != 480 {
		t.Errorf("size = %dx%d, want 640x480", got.Width, got.Height)
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
	page, err := fatalReport{}.page(container.NewConfig(container.Config{}), errFatalSample)
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
}

var errFatalSample = stringError("sample")

type stringError string

func (e stringError) Error() string { return string(e) }

func TestFatalReportPage_DrawsNoPluginAction(t *testing.T) {
	page, err := fatalReport{}.page(container.NewConfig(container.Config{}), errFatalSample)
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
