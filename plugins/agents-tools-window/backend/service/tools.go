package service

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/agents-tools-window/backend/api"
	"github.com/jiva-studio/numen/plugins/agents-tools-window/backend/usecase/editpreview"
	"github.com/jiva-studio/numen/plugins/agents-tools-window/backend/usecase/preview"
	"github.com/jiva-studio/numen/plugins/agents-tools-window/backend/usecase/window"
)

// Host provides the vault and its window to the window tools. A nil field means the host offers
// none.
type Host struct {
	Vault  host.FileReader
	Window host.Window
}

// RegisterTools registers all window interaction and edit preview tools on the given registry.
func RegisterTools(registry *tools.Registry, previews *preview.Store, on Host) error {
	control := window.NewControl(on.Window)
	creator := editpreview.NewCreator(previews, on.Vault)
	for _, add := range []func(*tools.Registry) error{
		func(r *tools.Registry) error { return addWindowTabList(r, control) },
		func(r *tools.Registry) error { return addWindowTabActivate(r, control) },
		func(r *tools.Registry) error { return addNoteFocus(r, control) },
		func(r *tools.Registry) error { return addSourceFocus(r, control) },
		func(r *tools.Registry) error { return addNoteEditPreview(r, creator) },
	} {
		if err := add(registry); err != nil {
			return err
		}
	}
	return nil
}

func addWindowTabList(registry *tools.Registry, control window.Control) error {
	return tools.Add(registry, tools.Spec{
		Name:  "window_tab_list",
		Title: "List open window tabs",
		Description: "List the open tabs of the window with their titles, paths, kinds and places, " +
			"and the identifier of the active tab. The place is the one the reader of the tab reported, " +
			"omitted for a tab with none. Call it before describing the active tab.",
		Effect:       tools.ReadOnly,
		Instructions: []string{tabListSentence},
		Requires:     []string{host.PermissionWindowFocus},
	}, func(ctx context.Context, _ api.WindowTabListInput) (api.WindowTabListOutput, error) {
		open, activeID, err := control.ListTabs(ctx, tools.GetCallContext(ctx)["focus"])
		if err != nil {
			return api.WindowTabListOutput{}, err
		}
		return api.WindowTabListOutput{Tabs: newTabInfos(open, activeID), ActiveTabID: activeID}, nil
	})
}

func addNoteEditPreview(registry *tools.Registry, creator editpreview.Creator) error {
	return tools.Add(registry, tools.Spec{
		Name:  "note_edit_preview",
		Title: "Preview note edit",
		Description: "Propose changes to a note with an interactive diff preview in the editor " +
			"without directly modifying the vault file.",
		Effect: tools.ReadOnly,
	}, func(ctx context.Context, in api.NoteEditPreviewInput) (api.NoteEditPreviewOutput, error) {
		created, err := creator.Create(ctx, editpreview.Request{
			Path:         in.Path,
			ProposedText: in.ProposedText,
			OriginalText: in.OriginalText,
			Diff:         in.Diff,
		})
		if err != nil {
			return api.NoteEditPreviewOutput{}, err
		}
		return api.NoteEditPreviewOutput{
			PreviewID: created.ID,
			Path:      created.Path,
			Diff:      created.Diff,
			Additions: created.Additions,
			Deletions: created.Deletions,
		}, nil
	})
}

func newTabInfos(open []host.OpenTab, activeID string) []api.TabInfo {
	tabs := make([]api.TabInfo, 0, len(open))
	for _, tab := range open {
		tabs = append(tabs, newTabInfo(tab, activeID))
	}
	return tabs
}

func newTabInfo(tab host.OpenTab, activeID string) api.TabInfo {
	return api.TabInfo{
		TabID:    tab.ID,
		Kind:     tab.Kind,
		Path:     tab.Path,
		Title:    tab.Title,
		Place:    newPlace(tab.Place),
		IsActive: tab.ID != "" && tab.ID == activeID,
	}
}

func newPlace(place *host.TabPlace) *api.Place {
	if place == nil || *place == (host.TabPlace{}) {
		return nil
	}
	return &api.Place{
		Page:            place.Page,
		PageCount:       place.PageCount,
		PositionSeconds: place.PositionSeconds,
		CFI:             place.CFI,
		Chapter:         place.Chapter,
	}
}
