package service

import (
	"context"
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
)

func addNoteFocus(registry *tools.Registry, on Host) error {
	return tools.Add(registry, tools.Spec{
		Name:  "note_focus",
		Title: "Focus a note",
		Description: "Make a note the current note of the window; the neighborhood view is centered " +
			"on it. Call it only when the user asked to be taken to a note. Talking about one is " +
			"not asking, and neither is editing one: the call changes what the window shows. It " +
			"returns no note content: read the note for its content.",
		Effect:       tools.ReadOnly,
		Step:         tools.ByEffect,
		Servers:      []tools.Server{tools.Full},
		Instructions: []string{showSentence},
		Requires:     []string{host.PermissionWindowFocus},
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
			"Pass the start and length of each passage unchanged, as a search of the library returns them. " +
			"When several passages of one source are relevant, pass them all; the window " +
			"scrolls to the first. An empty list opens the file at its beginning. Call it only when " +
			"the user asked to see a passage. Talking about a passage is not asking: the call " +
			"changes what the window shows.",
		Effect:       tools.ReadOnly,
		Step:         tools.ByEffect,
		Servers:      []tools.Server{tools.Full},
		Instructions: []string{showSentence, passagesSentence},
		Requires:     []string{host.PermissionWindowFocus},
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

func addWindowTabActivate(registry *tools.Registry, on Host) error {
	return tools.Add(registry, tools.Spec{
		Name:  "window_tab_activate",
		Title: "Activate window tab",
		Description: "Make the open tab with the id the active tab of the window. Use an id from " +
			"`window_tab_list`. Call it only when the user asked for that tab: the call changes what " +
			"the window shows.",
		Effect:       tools.ReadOnly,
		Step:         tools.ByEffect,
		Servers:      []tools.Server{tools.Full},
		Instructions: []string{showSentence, tabIDSentence},
		Requires:     []string{host.PermissionWindowFocus},
	}, func(ctx context.Context, in WindowTabActivateInput) (WindowTabActivateOutput, error) {
		if in.TabID == "" {
			return WindowTabActivateOutput{}, errMissingTabID
		}
		info, err := activateWindowTab(ctx, on, in.TabID)
		if err != nil {
			return WindowTabActivateOutput{}, err
		}
		return WindowTabActivateOutput{Tab: info}, nil
	})
}

func activateWindowTab(ctx context.Context, on Host, id string) (TabInfo, error) {
	if on.Window == nil {
		return TabInfo{}, host.ErrNoWindow
	}
	open, _, err := on.Window.ReadOpenTabs(ctx)
	if err != nil {
		return TabInfo{}, err
	}
	index := slices.IndexFunc(open, func(tab host.OpenTab) bool { return tab.ID == id })
	if index < 0 {
		return TabInfo{}, fmt.Errorf("no open tab has id %q", id)
	}
	tab, activeID, err := on.Window.ActivateTab(ctx, id)
	if errors.Is(err, host.ErrNotExist) {
		return TabInfo{}, fmt.Errorf("no open tab has id %q", id)
	}
	if errors.Is(err, host.ErrTabNotShown) {
		return TabInfo{}, fmt.Errorf("the window did not show tab %q within %s", id, tabActivationBound)
	}
	if err != nil {
		return TabInfo{}, err
	}
	return newTabInfo(tab, activeID), nil
}

func focusFile(ctx context.Context, on Host, path string, passages []host.PassageSpan) error {
	if on.Window == nil {
		return host.ErrNoWindow
	}
	return on.Window.FocusPassages(ctx, path, passages)
}

func isNotePath(filePath string) bool {
	return strings.EqualFold(path.Ext(filePath), domain.NoteExtension)
}

// tabActivationBound is the time the host waits for the window to show an activated tab. The host
// does not send its bound, so the text of the error names the default of the host.
const tabActivationBound = "2 s"
