package service

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
)

func TestFindOpenTabPrefersTheIDToThePathAndThePathToTheTitle(t *testing.T) {
	open := []host.OpenTab{
		{ID: "tab-1", Path: "notes/Heron.md", Title: "Bird Guide"},
		{ID: "tab-2", Path: "library/Bird Guide.pdf", Title: "Grey Heron"},
	}

	if tab, isFound := findOpenTab(open, WindowFocusTabInput{TabID: "tab-2", Path: "notes/Heron.md", Title: "Bird Guide"}); !isFound || tab.ID != "tab-2" {
		t.Errorf("an id and a path of another tab found %+v (%v), want tab-2", tab, isFound)
	}
	if tab, isFound := findOpenTab(open, WindowFocusTabInput{TabID: "", Path: "notes/Heron.md", Title: "Grey Heron"}); !isFound || tab.ID != "tab-1" {
		t.Errorf("a path and a title of another tab found %+v (%v), want tab-1", tab, isFound)
	}
}
