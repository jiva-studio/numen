package service

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/agents-tools-window/backend/internal/adapter/clock"
)

// RegisterTools registers all window interaction and edit preview tools on the given registry.
func RegisterTools(registry *tools.Registry, manager *WindowManager, vault host.Vault) error {
	for _, add := range []func(*tools.Registry) error{
		func(r *tools.Registry) error { return addWindowTabsGet(r, manager) },
		func(r *tools.Registry) error { return addWindowFocusTab(r, manager) },
		func(r *tools.Registry) error { return addEditPreview(r, manager, vault) },
	} {
		if err := add(registry); err != nil {
			return err
		}
	}
	return nil
}

func addWindowTabsGet(registry *tools.Registry, manager *WindowManager) error {
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
		tabs, activeID := manager.GetOpenTabs()
		requestContext := tools.GetCallContext(ctx)

		if len(tabs) == 0 && requestContext != nil {
			if focus := requestContext["focus"]; focus != "" {
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

		summary := formatTabsSummary(tabs, activeID)
		return WindowTabsGetOutput{
			Tabs:     tabs,
			ActiveID: activeID,
			Summary:  summary,
		}, nil
	})
}

func addWindowFocusTab(registry *tools.Registry, manager *WindowManager) error {
	return tools.Add(registry, tools.Spec{
		Name:  "window_focus_tab",
		Title: "Focus window tab",
		Description: "Switch or focus a specific tab or note in the user window by tab ID, " +
			"path, or title. Moves the window focus to the requested tab.",
		Effect:  tools.ReadOnly,
		Step:    tools.ByEffect,
		Servers: []tools.Server{tools.Full, tools.Reading},
	}, func(ctx context.Context, in WindowFocusTabInput) (WindowFocusTabOutput, error) {
		tab, err := manager.FocusTab(ctx, in)
		if err != nil {
			return WindowFocusTabOutput{}, err
		}
		msg := fmt.Sprintf("focused tab %s", formatTabName(tab))
		return WindowFocusTabOutput{
			Success: true,
			Tab:     tab,
			Message: msg,
		}, nil
	})
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
			CreatedAt:       clock.Now(),
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
