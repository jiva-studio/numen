package service

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
)

// Host is what the window tools work through: the vault, its window, and the text versions of its
// files. A nil field is a host that offers none.
type Host struct {
	Vault  host.Vault
	Window host.Window
	Texts  host.Texts
}

// RegisterTools registers all window interaction and edit preview tools on the given registry.
func RegisterTools(registry *tools.Registry, manager *WindowManager, on Host) error {
	for _, add := range []func(*tools.Registry) error{
		func(r *tools.Registry) error { return addWindowTabsGet(r, on) },
		func(r *tools.Registry) error { return addWindowFocusTab(r, on) },
		func(r *tools.Registry) error { return addNoteFocus(r, on) },
		func(r *tools.Registry) error { return addSourceFocus(r, on) },
		func(r *tools.Registry) error { return addEditPreview(r, manager, on.Vault) },
	} {
		if err := add(registry); err != nil {
			return err
		}
	}
	return nil
}

func addWindowTabsGet(registry *tools.Registry, on Host) error {
	return tools.Add(registry, tools.Spec{
		Name:  "window_tabs_get",
		Title: "Get open window tabs",
		Description: "Inspect currently open tabs in the user window, including their titles, " +
			"paths, kinds, positions, and which tab is in front. Ask it before stating " +
			"what is in front of the user.",
		Effect:  tools.ReadOnly,
		Step:    tools.ByEffect,
		Servers: []tools.Server{tools.Full, tools.Reading},
	}, func(ctx context.Context, _ WindowTabsGetInput) (WindowTabsGetOutput, error) {
		tabs, activeID, err := readWindowTabs(ctx, on)
		if err != nil {
			return WindowTabsGetOutput{}, err
		}
		if len(tabs) == 0 {
			if focus := tools.GetCallContext(ctx)["focus"]; focus != "" {
				inferredTab := TabInfo{
					ID:       "front-tab",
					Kind:     "",
					Path:     focus,
					Title:    formatTitleFromPath(focus),
					Where:    "",
					IsActive: true,
				}
				tabs = []TabInfo{inferredTab}
				activeID = inferredTab.ID
			}
		}
		return WindowTabsGetOutput{
			Tabs:     tabs,
			ActiveID: activeID,
			Summary:  formatTabsSummary(tabs, activeID),
		}, nil
	})
}

// readWindowTabs is the tabs the window last reported, each with the place it is open at, and the
// active one. A host with no window has none.
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
	texts := getCallTexts(ctx, on.Texts)
	tabs := make([]TabInfo, 0, len(open))
	for _, tab := range open {
		tabs = append(tabs, TabInfo{
			ID:       tab.ID,
			Kind:     tab.Kind,
			Path:     tab.Path,
			Title:    tab.Title,
			Where:    readPosition(ctx, texts, tab),
			IsActive: tab.ID != "" && tab.ID == activeID,
		})
	}
	return tabs, activeID, nil
}

func addEditPreview(registry *tools.Registry, manager *WindowManager, vault host.Vault) error {
	return tools.Add(registry, tools.Spec{
		Name:  "edit_preview",
		Title: "Preview note edit",
		Description: "Propose changes to a note with an interactive diff preview in the editor " +
			"without directly modifying the vault file.",
		Effect:  tools.ReadOnly,
		Step:    tools.ByEffect,
		Servers: []tools.Server{tools.Full, tools.Reading},
	}, func(ctx context.Context, in EditPreviewInput) (EditPreviewOutput, error) {
		if in.Path == "" {
			return EditPreviewOutput{}, errors.New("path is required")
		}
		if in.ProposedContent == "" && in.Diff == "" {
			return EditPreviewOutput{}, errors.New("either proposedContent or diff is required")
		}

		diffText, additions, deletions := prepareDiff(ctx, in, vault)

		previewID := manager.NextPreviewID()
		preview := EditPreview{
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

		summary := fmt.Sprintf("proposed changes to %s: +%d, -%d lines", in.Path, additions, deletions)
		return EditPreviewOutput{
			PreviewID: previewID,
			Path:      in.Path,
			Diff:      diffText,
			Additions: additions,
			Deletions: deletions,
			Status:    "preview_ready",
			Summary:   summary,
		}, nil
	})
}

func prepareDiff(ctx context.Context, in EditPreviewInput, vault host.Vault) (string, int, int) {
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

func formatTabsSummary(tabs []TabInfo, activeID string) string {
	if len(tabs) == 0 {
		return "the window has nothing open"
	}
	var activeTab *TabInfo
	for i := range tabs {
		if tabs[i].ID != "" && tabs[i].ID == activeID {
			activeTab = &tabs[i]
			break
		}
		if tabs[i].IsActive {
			activeTab = &tabs[i]
		}
	}
	if activeTab == nil {
		activeTab = &tabs[0]
	}
	return describeTab(*activeTab)
}

func describeTab(t TabInfo) string {
	name := formatTabName(t)
	kind := t.Kind
	if kind == "" {
		kind = "tab"
	}
	if t.Title == "" && t.Path == "" {
		return fmt.Sprintf("a tab of kind %q is in front of them", kind)
	}
	if t.Where != "" {
		return fmt.Sprintf("the %s %s is in front of them, open at %s", kind, name, t.Where)
	}
	return fmt.Sprintf("the %s %s is in front of them", kind, name)
}

func formatTabName(t TabInfo) string {
	switch {
	case t.Title != "" && t.Path != "":
		return fmt.Sprintf("%q at %s", t.Title, t.Path)
	case t.Path != "":
		return t.Path
	}
	return fmt.Sprintf("%q", t.Title)
}

func formatTitleFromPath(filePath string) string {
	base := filepath.Base(filePath)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}
