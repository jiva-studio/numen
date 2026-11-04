package app

import (
	"runtime"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/adapter/window/editor"
	"github.com/jiva-studio/numen/libs/core/container"
)

// hasSystemFrame is whether the windows of this platform keep the system title bar.
func hasSystemFrame() bool { return runtime.GOOS == "darwin" }

func TestMainWindowDrawsItsOwnTitleBar(t *testing.T) {
	opened, _ := windowOn(t)

	got := getMainWindowOptions(opened)

	if got.Frameless == hasSystemFrame() {
		t.Errorf("frameless = %v on %s", got.Frameless, runtime.GOOS)
	}
	if got.Title == "" {
		t.Error("the window has no title for the taskbar")
	}
}

func TestSecondaryWindowDrawsItsOwnTitleBar(t *testing.T) {
	opened, _ := windowOn(t)

	got := getSecondaryWindowOptions(opened, editor.DetachedTab{ID: "a", X: 10, Y: 20})

	if got.Frameless == hasSystemFrame() {
		t.Errorf("frameless = %v on %s", got.Frameless, runtime.GOOS)
	}
	if got.X != 10 || got.Y != 20 {
		t.Errorf("position = %d,%d, want 10,20", got.X, got.Y)
	}
}

func TestFatalReportDrawsItsOwnTitleBar(t *testing.T) {
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

func TestFatalWindowIsFramelessWhereTheBarIsDrawn(t *testing.T) {
	got := getFatalWindowOptions()

	if got.Frameless == hasSystemFrame() {
		t.Errorf("frameless = %v on %s", got.Frameless, runtime.GOOS)
	}
}

var errFatalSample = errString("sample")

type errString string

func (e errString) Error() string { return string(e) }
