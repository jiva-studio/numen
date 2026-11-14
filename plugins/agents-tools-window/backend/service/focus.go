package service

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/agents-tools-window/backend/api"
	"github.com/jiva-studio/numen/plugins/agents-tools-window/backend/usecase/window"
)

func addNoteFocus(registry *tools.Registry, control window.Control) error {
	return tools.Add(registry, tools.Spec{
		Name:  "note_focus",
		Title: "Focus a note",
		Description: "Make a note the current note of the window; the neighborhood view is centered " +
			"on it. Call it only when the user asked to be taken to a note. Talking about one is " +
			"not asking, and neither is editing one: the call changes what the window shows. It " +
			"returns no note content: read the note for its content.",
		Effect:       tools.ReadOnly,
		Instructions: []string{showSentence},
		Requires:     []string{host.PermissionWindowFocus},
	}, func(ctx context.Context, in api.NoteFocusInput) (api.NoteFocusOutput, error) {
		note, err := control.FocusNote(ctx, in.Path)
		if err != nil {
			return api.NoteFocusOutput{}, err
		}
		return api.NoteFocusOutput{Note: api.FocusedNote{Path: note.Path, Title: note.Title, ID: note.ID}}, nil
	})
}

func addSourceFocus(registry *tools.Registry, control window.Control) error {
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
		Instructions: []string{showSentence, spansSentence},
		Requires:     []string{host.PermissionWindowFocus},
	}, func(ctx context.Context, in api.SourceFocusInput) (api.SourceFocusOutput, error) {
		spans := make([]host.ByteSpan, 0, len(in.Spans))
		for _, span := range in.Spans {
			spans = append(spans, host.ByteSpan{Start: span.Start, Length: span.Length})
		}
		highlightCount, err := control.FocusFile(ctx, in.Path, spans)
		if err != nil {
			return api.SourceFocusOutput{}, err
		}
		return api.SourceFocusOutput{Path: in.Path, HighlightCount: highlightCount}, nil
	})
}

func addWindowTabActivate(registry *tools.Registry, control window.Control) error {
	return tools.Add(registry, tools.Spec{
		Name:  "window_tab_activate",
		Title: "Activate window tab",
		Description: "Make the open tab with the id the active tab of the window. Use an id from " +
			"`window_tab_list`. Call it only when the user asked for that tab: the call changes what " +
			"the window shows.",
		Effect:       tools.ReadOnly,
		Instructions: []string{showSentence, tabIDSentence},
		Requires:     []string{host.PermissionWindowFocus},
	}, func(ctx context.Context, in api.WindowTabActivateInput) (api.WindowTabActivateOutput, error) {
		tab, activeID, err := control.ActivateTab(ctx, in.TabID)
		if err != nil {
			return api.WindowTabActivateOutput{}, err
		}
		return api.WindowTabActivateOutput{Tab: newTabInfo(tab, activeID)}, nil
	})
}
