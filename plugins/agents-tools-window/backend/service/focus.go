package service

import (
	"context"
	"errors"
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

func addWindowFocusTab(registry *tools.Registry, on Host) error {
	return tools.Add(registry, tools.Spec{
		Name:  "window_focus_tab",
		Title: "Focus window tab",
		Description: "Make a tab of the window the active tab, found by its tab ID, else its path, " +
			"else its title. A path no tab holds opens that note or file. Call it only when the " +
			"user asked to open the tab: the call changes what the window shows.",
		Effect:  tools.ReadOnly,
		Step:    tools.ByEffect,
		Servers: []tools.Server{tools.Full},
	}, func(context.Context, WindowFocusTabInput) (WindowFocusTabOutput, error) {
		return WindowFocusTabOutput{}, errors.New("window_focus_tab is not implemented")
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
