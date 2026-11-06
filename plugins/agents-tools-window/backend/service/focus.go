package service

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
)

func addNoteFocus(registry *tools.Registry, on Host) error {
	return tools.Add(registry, tools.Spec{
		Name:  "note_focus",
		Title: "Focus a note",
		Description: "Make a note the current note of the window; the neighbourhood view is drawn " +
			"around it. Call it only when the user asked to be taken to a note. Talking about one is " +
			"not asking, and neither is editing one: the call changes what the window shows. It " +
			"returns no note content: `note_read` does that.",
		Effect:  tools.ReadOnly,
		Step:    tools.ByEffect,
		Servers: []tools.Server{tools.Full},
	}, func(ctx context.Context, in NoteFocusInput) (NoteFocusOutput, error) {
		if on.Window == nil {
			return NoteFocusOutput{}, host.ErrNoWindow
		}
		note, err := on.Window.FocusNote(ctx, in.Path)
		if err != nil {
			return NoteFocusOutput{}, err
		}
		return NoteFocusOutput{Note: FocusedNote{Path: note.Path, Title: note.Title, ID: note.ID}}, nil
	})
}

func addSourceFocus(registry *tools.Registry, on Host) error {
	return tools.Add(registry, tools.Spec{
		Name:  "source_focus",
		Title: "Focus passages of a file",
		Description: "Open a file of the vault in the window, scroll to the first passage and " +
			"highlight every passage; a note opens whole with nothing highlighted. " +
			"`library_search` returns the start and length of each passage; pass them unchanged. " +
			"When several passages of one source answer a question, pass them all; the window " +
			"scrolls to the first. An empty list opens the file at its beginning. Call it only when " +
			"the user asked to see a passage. Talking about a passage is not asking: the call " +
			"changes what the window shows.",
		Effect:  tools.ReadOnly,
		Step:    tools.ByEffect,
		Servers: []tools.Server{tools.Full},
	}, func(ctx context.Context, in SourceFocusInput) (SourceFocusOutput, error) {
		passages := make([]host.PassageSpan, 0, len(in.Passages))
		for _, passage := range in.Passages {
			passages = append(passages, host.PassageSpan{Start: passage.Start, Length: passage.Length})
		}
		if err := focusFile(ctx, on, in.Path, passages); err != nil {
			return SourceFocusOutput{}, err
		}
		highlightCount := len(passages)
		if isNotePath(in.Path) {
			highlightCount = 0
		}
		return SourceFocusOutput{Path: in.Path, HighlightCount: highlightCount}, nil
	})
}

var errMissingTabID = errors.New("tabId is required")

func addWindowFocusTab(registry *tools.Registry, on Host) error {
	return tools.Add(registry, tools.Spec{
		Name:  "window_focus_tab",
		Title: "Focus window tab",
		Description: "Make the open tab with the id the active tab of the window. Take the id from " +
			"`window_tabs_get`. Call it only when the user asked for that tab: the call changes what " +
			"the window shows.",
		Effect:  tools.ReadOnly,
		Step:    tools.ByEffect,
		Servers: []tools.Server{tools.Full},
	}, func(ctx context.Context, in WindowFocusTabInput) (WindowFocusTabOutput, error) {
		if in.TabID == "" {
			return WindowFocusTabOutput{}, errMissingTabID
		}
		if on.Window == nil {
			return WindowFocusTabOutput{}, host.ErrNoWindow
		}
		tab, activeID, err := on.Window.ActivateTab(ctx, in.TabID)
		if errors.Is(err, host.ErrNotExist) {
			return WindowFocusTabOutput{}, fmt.Errorf("no open tab has id %q", in.TabID)
		}
		if err != nil {
			return WindowFocusTabOutput{}, err
		}
		info, err := newTabInfo(tab, activeID)
		if err != nil {
			return WindowFocusTabOutput{}, err
		}
		return WindowFocusTabOutput{Tab: info}, nil
	})
}

// focusFile opens the file in the window at the passages and highlights them.
func focusFile(ctx context.Context, on Host, path string, passages []host.PassageSpan) error {
	if on.Window == nil {
		return host.ErrNoWindow
	}
	return on.Window.FocusPassages(ctx, path, passages)
}

// isNotePath reports whether the path names a note of the vault.
func isNotePath(filePath string) bool {
	return strings.EqualFold(path.Ext(filePath), domain.NoteExtension)
}
