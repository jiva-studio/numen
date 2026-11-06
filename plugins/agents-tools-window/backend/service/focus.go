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

var (
	errMissingFocusTarget = errors.New("specify tabId, path, or title to focus")
	errTabNotFound        = errors.New("no open tab matches criteria")
	errTabWithoutPath     = errors.New("the tab holds no file to focus")
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
	}, func(ctx context.Context, in WindowFocusTabInput) (WindowFocusTabOutput, error) {
		tab, err := findFocusTab(ctx, on, in)
		if err != nil {
			return WindowFocusTabOutput{}, err
		}
		if err := focusPath(ctx, on, tab.Path); err != nil {
			return WindowFocusTabOutput{}, err
		}
		return WindowFocusTabOutput{Tab: tab}, nil
	})
}

// focusPath moves the window to the path: a note becomes the current one, and any other file opens
// at its beginning.
func focusPath(ctx context.Context, on Host, filePath string) error {
	if !isNotePath(filePath) {
		return focusFile(ctx, on, filePath, nil)
	}
	if on.Window == nil {
		return host.ErrNoWindow
	}
	_, err := on.Window.FocusNote(ctx, filePath)
	return err
}

// focusFile opens the file in the window at the passages and highlights them.
func focusFile(ctx context.Context, on Host, path string, passages []host.PassageSpan) error {
	if on.Window == nil {
		return host.ErrNoWindow
	}
	return on.Window.FocusPassages(ctx, path, passages)
}

// findFocusTab is the open tab the input names, by its id, else its path, else its title, or a tab
// of the path where no open tab matches. A tab that holds no file is an error.
func findFocusTab(ctx context.Context, on Host, in WindowFocusTabInput) (TabInfo, error) {
	if in.TabID == "" && in.Path == "" && in.Title == "" {
		return TabInfo{}, errMissingFocusTarget
	}
	if on.Window == nil {
		return TabInfo{}, host.ErrNoWindow
	}
	open, _, err := on.Window.ReadOpenTabs(ctx)
	if err != nil {
		return TabInfo{}, err
	}
	tab, isFound := findOpenTab(open, in)
	switch {
	case isFound && tab.Path == "":
		return TabInfo{}, errTabWithoutPath
	case isFound:
		return TabInfo{ID: tab.ID, Kind: tab.Kind, Path: tab.Path, Title: tab.Title, Location: "", IsActive: true}, nil
	case in.Path != "":
		return TabInfo{ID: "", Kind: "", Path: in.Path, Title: formatTitleFromPath(in.Path), Location: "", IsActive: true}, nil
	}
	return TabInfo{}, errTabNotFound
}

// findOpenTab is the tab whose id the input names, else whose path, else whose title.
func findOpenTab(open []host.OpenTab, in WindowFocusTabInput) (host.OpenTab, bool) {
	for _, matches := range []func(host.OpenTab) bool{
		func(tab host.OpenTab) bool { return in.TabID != "" && tab.ID == in.TabID },
		func(tab host.OpenTab) bool { return in.Path != "" && tab.Path == in.Path },
		func(tab host.OpenTab) bool { return in.Title != "" && tab.Title == in.Title },
	} {
		for _, tab := range open {
			if matches(tab) {
				return tab, true
			}
		}
	}
	return host.OpenTab{}, false
}

// isNotePath reports whether the path names a note of the vault.
func isNotePath(filePath string) bool {
	return strings.EqualFold(path.Ext(filePath), domain.NoteExtension)
}
