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

func TestApplyHostsTheWindowsPageForTheSystemButtonRegions(t *testing.T) {
	if !Apply(application.WebviewWindowOptions{}, "windows").Windows.WebView2CompositionHosting {
		t.Error("the Windows window does not map the page's window buttons to the system buttons")
	}
	if Apply(application.WebviewWindowOptions{}, "linux").Windows.WebView2CompositionHosting {
		t.Error("the Linux window sets an option of Windows")
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
}

func TestApplyKeepsNoNativeDragAreaOnMac(t *testing.T) {
	got := Apply(application.WebviewWindowOptions{}, "darwin")

	if got.Mac.InvisibleTitleBarHeight != 0 {
		t.Errorf("invisible title bar height = %d; the tabs at the top would move the window", got.Mac.InvisibleTitleBarHeight)
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
