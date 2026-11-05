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

var (
	errMissingFocusTarget = errors.New("specify tabId, path, or title to focus")
	errTabNotFound        = errors.New("no open tab matches criteria")
	errTabWithoutPath     = errors.New("the tab holds no file to focus")
)

func addNoteFocus(registry *tools.Registry, on Host) error {
	return tools.Add(registry, tools.Spec{
		Name:  "note_focus",
		Title: "Focus a note",
		Description: "Make a note the one the person is looking at, so that the " +
			"neighbourhood they see is drawn around it. Use it only where the person " +
			"asked to be taken to a note. Talking about one is not asking, and neither " +
			"is changing one: the person may be working somewhere else, and this moves " +
			"them. It reads nothing back: `note_read` does that.",
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
		return NoteFocusOutput{Focused: FocusedNote{Path: note.Path, Title: note.Title, ID: note.ID}}, nil
	})
}

func addSourceFocus(registry *tools.Registry, on Host) error {
	return tools.Add(registry, tools.Spec{
		Name:  "source_focus",
		Title: "Focus passages of a file",
		Description: "Open a file of the vault in front of the person: the window goes to the " +
			"first passage and lights all of them. `library_search` gives the start and " +
			"length of every passage it answers with; pass them unchanged. Where a question " +
			"is answered in several places of one source, pass them all: the first is the one " +
			"the person is taken to. No passages opens the file at its beginning. Use it only " +
			"where the person asked to be shown something. Talking about a passage is not " +
			"asking: the person may be working somewhere else, and this moves them.",
		Effect:  tools.ReadOnly,
		Step:    tools.ByEffect,
		Servers: []tools.Server{tools.Full},
	}, func(ctx context.Context, in SourceFocusInput) (SourceFocusOutput, error) {
		passages := make([]host.PassageSpan, 0, len(in.Passages))
		for _, passage := range in.Passages {
			passages = append(passages, host.PassageSpan{Start: passage.Start, Length: passage.Length})
		}
		summary, err := focusFile(ctx, on, in.Path, passages)
		if err != nil {
			return SourceFocusOutput{}, err
		}
		return SourceFocusOutput{IsShown: true, Summary: summary}, nil
	})
}

func addWindowFocusTab(registry *tools.Registry, on Host) error {
	return tools.Add(registry, tools.Spec{
		Name:  "window_focus_tab",
		Title: "Focus window tab",
		Description: "Bring a tab of the person's window to the front, found by its tab ID, else " +
			"its path, else its title. A path no tab holds opens that note or file. Use it only " +
			"where the person asked to be taken there: this moves them.",
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
		return WindowFocusTabOutput{
			Success: true,
			Tab:     tab,
			Message: fmt.Sprintf("focused tab %s", formatTabName(tab)),
		}, nil
	})
}

// focusPath moves the window to the path: a note becomes the current one, and any other file opens
// at its beginning.
func focusPath(ctx context.Context, on Host, filePath string) error {
	if !isNotePath(filePath) {
		_, err := focusFile(ctx, on, filePath, nil)
		return err
	}
	if on.Window == nil {
		return host.ErrNoWindow
	}
	_, err := on.Window.FocusNote(ctx, filePath)
	return err
}

// focusFile opens the file in the window at the passages and returns what the person now looks at.
func focusFile(ctx context.Context, on Host, path string, passages []host.PassageSpan) (string, error) {
	if on.Window == nil {
		return "", host.ErrNoWindow
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
		return TabInfo{ID: tab.ID, Kind: tab.Kind, Path: tab.Path, Title: tab.Title, Where: "", IsActive: true}, nil
	case in.Path != "":
		return TabInfo{ID: "", Kind: "", Path: in.Path, Title: formatTitleFromPath(in.Path), Where: "", IsActive: true}, nil
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
