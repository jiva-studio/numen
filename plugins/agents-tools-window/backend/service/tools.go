package service

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
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
		Effect:  tools.Reads,
		Servers: []tools.Server{tools.Full, tools.Reading, tools.Review},
	}, func(ctx context.Context, _ WindowTabsGetInput) (WindowTabsGetOutput, error) {
		tabs, frontID := manager.GetOpenTabs()
		requestContext := tools.Asked(ctx)

		if len(tabs) == 0 && requestContext != nil {
			if focus := requestContext["focus"]; focus != "" {
				inferredTab := TabInfo{
					ID:      "front-tab",
					Kind:    "note",
					Path:    focus,
					Title:   formatTitleFromPath(focus),
					IsFront: true,
				}
				tabs = []TabInfo{inferredTab}
				frontID = inferredTab.ID
			}
		}

		summary := formatTabsSummary(tabs, frontID)
		return WindowTabsGetOutput{
			Tabs:    tabs,
			FrontID: frontID,
			Summary: summary,
		}, nil
	})
}

func addWindowFocusTab(registry *tools.Registry, manager *WindowManager) error {
	return tools.Add(registry, tools.Spec{
		Name:  "window_focus_tab",
		Title: "Focus window tab",
		Description: "Switch or focus a specific tab or note in the user window by tab ID, " +
			"path, or title. Moves the window focus to the requested tab.",
		Effect:  tools.Reads,
		Servers: []tools.Server{tools.Full, tools.Reading, tools.Review},
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
		Effect:  tools.Reads,
		Servers: []tools.Server{tools.Full, tools.Reading, tools.Review},
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
			CreatedAt:       time.Now(),
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

func formatTabsSummary(tabs []TabInfo, frontID string) string {
	if len(tabs) == 0 {
		return "the window has nothing open"
	}
	var frontTab *TabInfo
	for i := range tabs {
		if tabs[i].ID != "" && tabs[i].ID == frontID {
			frontTab = &tabs[i]
			break
		}
		if tabs[i].IsFront {
			frontTab = &tabs[i]
		}
	}
	if frontTab == nil {
		frontTab = &tabs[0]
	}
	return describeTab(*frontTab)
}

func describeTab(t TabInfo) string {
	name := formatTabName(t)
	switch t.Kind {
	case "note":
		return fmt.Sprintf("the note %s is in front of them", name)
	case "plex":
		if t.Path == "" {
			return "they are looking at a plex standing on no note"
		}
		return fmt.Sprintf("they are looking at the plex around the note %s", name)
	case "document":
		if t.Where != "" {
			return fmt.Sprintf("the document %s is in front of them, open at %s", name, t.Where)
		}
		return fmt.Sprintf("the document %s is in front of them", name)
	case "book":
		if t.Where != "" {
			return fmt.Sprintf("the book %s is in front of them, open at %s", name, t.Where)
		}
		return fmt.Sprintf("the book %s is in front of them", name)
	case "recording":
		if t.Where != "" {
			return fmt.Sprintf("the recording %s is in front of them, with %s", name, t.Where)
		}
		return fmt.Sprintf("the recording %s is in front of them", name)
	default:
		if t.Kind != "" {
			return fmt.Sprintf("a tab of kind %q is in front of them", t.Kind)
		}
		return fmt.Sprintf("the tab %s is in front of them", name)
	}
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
