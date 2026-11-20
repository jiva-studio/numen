package opentabs

import (
	"encoding/json"

	windowdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/window"
)

// workspace is the JSON value of the editor.workspace entry.
type workspace struct {
	Tabs     []tab           `json:"tabs"`
	ActiveID string          `json:"activeId"`
	Layout   json.RawMessage `json:"layout,omitempty"`
}

// tab holds the fields restored when the vault opens.
type tab struct {
	ID    string `json:"id"`
	Kind  string `json:"kind"`
	Path  string `json:"path"`
	Title string `json:"title"`
	State string `json:"state"`
}

func newWorkspace(open windowdomain.OpenTabs) workspace {
	saved := workspace{Tabs: make([]tab, len(open.Tabs)), Layout: json.RawMessage(open.Layout)}
	for position, one := range open.Tabs {
		saved.Tabs[position] = tab{ID: one.ID, Kind: one.Kind, Path: one.Path, Title: one.Title, State: one.State}
		if one.ID == open.ActiveID {
			saved.ActiveID = one.ID
		}
	}
	return saved
}

func (w workspace) getOpenTabs() windowdomain.OpenTabs {
	open := windowdomain.OpenTabs{ActiveID: w.ActiveID, Layout: string(w.Layout)}
	for _, saved := range w.Tabs {
		open.Tabs = append(open.Tabs, windowdomain.Tab{
			ID: saved.ID, Kind: saved.Kind, Path: saved.Path, Title: saved.Title, State: saved.State,
		})
	}
	return open
}
