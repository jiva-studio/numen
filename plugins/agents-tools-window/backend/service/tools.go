package service

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
)

// Host provides the vault and its window to the window tools. A nil field means the host offers
// none.
type Host struct {
	Vault  host.Vault
	Window host.Window
}

// RegisterTools registers all window interaction and edit preview tools on the given registry.
func RegisterTools(registry *tools.Registry, manager *WindowManager, on Host) error {
	for _, add := range []func(*tools.Registry) error{
		func(r *tools.Registry) error { return addWindowTabList(r, on) },
		func(r *tools.Registry) error { return addWindowTabActivate(r, on) },
		func(r *tools.Registry) error { return addNoteFocus(r, on) },
		func(r *tools.Registry) error { return addSourceFocus(r, on) },
		func(r *tools.Registry) error { return addNoteEditPreview(r, manager, on.Vault) },
	} {
		if err := add(registry); err != nil {
			return err
		}
	}
	return nil
}

func addWindowTabList(registry *tools.Registry, on Host) error {
	return tools.Add(registry, tools.Spec{
		Name:  "window_tab_list",
		Title: "List open window tabs",
		Description: "List the open tabs of the window with their titles, paths, kinds and places, " +
			"and the identifier of the active tab. The place is the one the reader of the tab reported, " +
			"omitted for a tab with none. Call it before describing the active tab.",
		Effect:       tools.ReadOnly,
		Step:         tools.ByEffect,
		Servers:      []tools.Server{tools.Full, tools.Reading},
		Instructions: []string{tabListSentence},
		Requires:     []string{host.PermissionWindowFocus},
	}, func(ctx context.Context, _ WindowTabListInput) (WindowTabListOutput, error) {
		tabs, activeID, err := readWindowTabs(ctx, on)
		if err != nil {
			return WindowTabListOutput{}, err
		}
		if len(tabs) == 0 {
			if focus := tools.GetCallContext(ctx)["focus"]; focus != "" {
				inferredTab := TabInfo{
					TabID:    "front-tab",
					Kind:     "",
					Path:     focus,
					Title:    formatTitleFromPath(focus),
					Place:    nil,
					IsActive: true,
				}
				tabs = []TabInfo{inferredTab}
				activeID = inferredTab.TabID
			}
		}
		return WindowTabListOutput{
			Tabs:        tabs,
			ActiveTabID: activeID,
		}, nil
	})
}

// readWindowTabs returns the tabs the window last reported, each with the place it is open at, and
// the active one. A host with no window has none.
func readWindowTabs(ctx context.Context, on Host) ([]TabInfo, string, error) {
	if on.Window == nil {
		return nil, "", nil
	}
	open, activeID, err := on.Window.ReadOpenTabs(ctx)
	if errors.Is(err, host.ErrNoWindow) {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", err
	}
	tabs := make([]TabInfo, 0, len(open))
	for _, tab := range open {
		tabs = append(tabs, newTabInfo(tab, activeID))
	}
	return tabs, activeID, nil
}

// newTabInfo converts a tab to its tool output, with the place its reader reported. The tab is
// active when its id is the active id.
func newTabInfo(tab host.OpenTab, activeID string) TabInfo {
	return TabInfo{
		TabID:    tab.ID,
		Kind:     tab.Kind,
		Path:     tab.Path,
		Title:    tab.Title,
		Place:    newPlace(tab.Place),
		IsActive: tab.ID != "" && tab.ID == activeID,
	}
}

// newPlace converts the place a reader reported to its tool output, nil for a tab whose reader
// reported none.
func newPlace(place *host.TabPlace) *Place {
	if place == nil || *place == (host.TabPlace{}) {
		return nil
	}
	return &Place{
		Page:            place.Page,
		PageCount:       place.PageCount,
		PositionSeconds: place.PositionSeconds,
		CFI:             place.CFI,
		Chapter:         place.Chapter,
	}
}

func addNoteEditPreview(registry *tools.Registry, manager *WindowManager, vault host.Vault) error {
	return tools.Add(registry, tools.Spec{
		Name:  "note_edit_preview",
		Title: "Preview note edit",
		Description: "Propose changes to a note with an interactive diff preview in the editor " +
			"without directly modifying the vault file.",
		Effect:  tools.ReadOnly,
		Step:    tools.ByEffect,
		Servers: []tools.Server{tools.Full, tools.Reading},
	}, func(ctx context.Context, in NoteEditPreviewInput) (NoteEditPreviewOutput, error) {
		if in.Path == "" {
			return NoteEditPreviewOutput{}, errors.New("path is required")
		}
		if in.ProposedContent == "" && in.Diff == "" {
			return NoteEditPreviewOutput{}, errors.New("either proposedContent or diff is required")
		}

		diffText, additions, deletions := prepareDiff(ctx, in, vault)

		previewID := manager.NextPreviewID()
		preview := NoteEditPreview{
			ID:              previewID,
			Path:            in.Path,
			ProposedContent: in.ProposedContent,
			OriginalContent: in.OriginalContent,
			Diff:            diffText,
			Additions:       additions,
			Deletions:       deletions,
			CreatedAt:       manager.GetNow(),
		}
		manager.SavePreview(preview)

		return NoteEditPreviewOutput{
			PreviewID: previewID,
			Path:      in.Path,
			Diff:      diffText,
			Additions: additions,
			Deletions: deletions,
		}, nil
	})
}

func prepareDiff(ctx context.Context, in NoteEditPreviewInput, vault host.Vault) (string, int, int) {
	if in.Diff != "" {
		additions, deletions := CountDiffChanges(in.Diff)
		return in.Diff, additions, deletions
	}

	original := in.OriginalContent
	if original == "" && vault != nil {
		data, _, err := vault.Read(ctx, in.Path)
		if err == nil {
			original = string(data)
		}
	}
	return CreateUnifiedDiff(in.Path, original, in.ProposedContent)
}

func formatTitleFromPath(filePath string) string {
	base := filepath.Base(filePath)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}
