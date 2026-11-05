package frame

import (
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func TestApplyDrawsNoSystemFrameOnWindowsAndLinux(t *testing.T) {
	for _, goos := range []string{"windows", "linux"} {
		t.Run(goos, func(t *testing.T) {
			got := Apply(application.WebviewWindowOptions{Title: "numen"}, goos)

			if !got.Frameless {
				t.Error("the window keeps the system frame")
			}
			if got.Mac.TitleBar != (application.MacTitleBar{}) {
				t.Errorf("a mac title bar is set on %s: %+v", goos, got.Mac.TitleBar)
			}
		})
	}
}

func TestApplyKeepsTheSystemButtonsOnMac(t *testing.T) {
	got := Apply(application.WebviewWindowOptions{Title: "numen"}, "darwin")

	if got.Frameless {
		t.Error("the mac window is frameless; the system buttons would go")
	}
	if got.Mac.TitleBar != application.MacTitleBarHidden {
		t.Errorf("title bar = %+v, want the hidden bar", got.Mac.TitleBar)
	}
	if got.Mac.InvisibleTitleBarHeight != BarHeight {
		t.Errorf("invisible title bar height = %d, want %d", got.Mac.InvisibleTitleBarHeight, BarHeight)
	}
}

func TestApplyKeepsTheRestOfTheOptions(t *testing.T) {
	in := application.WebviewWindowOptions{Title: "numen — a", Width: 1280, Height: 860, URL: "/", EnableFileDrop: true}
	for _, goos := range []string{"windows", "linux", "darwin"} {
		got := Apply(in, goos)
		if got.Title != in.Title || got.Width != in.Width || got.Height != in.Height ||
			got.URL != in.URL || !got.EnableFileDrop {
			t.Errorf("%s: options changed: %+v", goos, got)
		}
	}
}
