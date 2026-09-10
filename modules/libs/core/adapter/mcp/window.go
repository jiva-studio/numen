package mcp

import (
	"context"
	"fmt"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/transcript"
)

// A Tab is one tab of the person's window, as an agent is told about it.
type Tab struct {
	Kind  string `json:"kind" jsonschema:"what sort of tab it is, in the window's own word for it"`
	Path  string `json:"path,omitempty" jsonschema:"the file it holds, by the path the vault files it under"`
	Title string `json:"title,omitempty" jsonschema:"what the tab is called, as the person reads it"`
	Where string `json:"where,omitempty" jsonschema:"where in that file the person stands"`
	Front bool   `json:"front,omitempty" jsonschema:"set on the one tab the person is looking at"`
}

// addWindowTools tells an agent what the person has open.
//
// They are added only where a window says. A binary nobody is sitting at
// answers about a vault and about nothing in front of anybody.
func addWindowTools(server *sdk.Server, core Core) {
	if core.Attending == nil {
		return
	}

	sdk.AddTool(server, &sdk.Tool{
		Name:  "window_tab_list",
		Title: "List open tabs",
		Description: "Every tab of the person's window, and which of them they are looking " +
			"at. Ask it before saying anything about what is open or in front of them: a " +
			"person moves between tabs while you work, so what they are looking at now is " +
			"not what they were looking at when the conversation began. A tab holding a " +
			"file names it by the path the vault files it under, and a tab whose kind you " +
			"do not know is a tab of that kind and nothing more. What a recording says is " +
			"read with source_read, as a document is.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, _ struct{}) (*sdk.CallToolResult, struct {
		Tabs    []Tab  `json:"tabs"`
		Looking string `json:"looking" jsonschema:"the tab the person is looking at, in words to say back to them"`
	}, error) {
		type out = struct {
			Tabs    []Tab  `json:"tabs"`
			Looking string `json:"looking" jsonschema:"the tab the person is looking at, in words to say back to them"`
		}
		open := core.Attending()
		res := out{Tabs: make([]Tab, 0, len(open.Tabs)), Looking: "the window has nothing open"}
		for _, one := range open.Tabs {
			front := one.ID != "" && one.ID == open.FrontID
			res.Tabs = append(res.Tabs, Tab{
				Kind:  one.Kind,
				Path:  one.Path,
				Title: one.Title,
				Where: tabPosition(one),
				Front: front,
			})
			if front {
				res.Looking = describeTab(one)
			}
		}
		return nil, res, nil
	})
}

// tabPresenter formats where in a tab the person is and how to describe it.
type tabPresenter struct {
	// position reports where in the tab the person is.
	position func(t domain.Tab) string
	// describe formats the tab description for an agent.
	describe func(t domain.Tab, name, where string) string
}

// tabPresenters maps tab kinds to their presenters.
var tabPresenters = map[string]tabPresenter{
	domain.TabNote: {
		describe: func(t domain.Tab, name, _ string) string {
			return fmt.Sprintf("the note %s is in front of them", name)
		},
	},
	domain.TabPlex: {
		describe: func(t domain.Tab, name, _ string) string {
			if t.Path == "" {
				return "they are looking at a plex standing on no note"
			}
			return fmt.Sprintf("they are looking at the plex around the note %s", name)
		},
	},
	domain.TabDocument: {
		position: documentPosition,
		describe: describeWithPosition("document"),
	},
	domain.TabBook: {
		position: bookPosition,
		describe: describeWithPosition("book"),
	},
	domain.TabRecording: {
		position: recordingPosition,
		describe: func(t domain.Tab, name, where string) string {
			return fmt.Sprintf("the recording %s is in front of them, with %s", name, where)
		},
	},
}

// tabPosition returns the position string for the given tab.
func tabPosition(t domain.Tab) string {
	presenter, known := tabPresenters[t.Kind]
	if !known || presenter.position == nil {
		return ""
	}
	return presenter.position(t)
}

// describeTab returns the description of the tab currently in front.
func describeTab(t domain.Tab) string {
	presenter, known := tabPresenters[t.Kind]
	if !known {
		return fmt.Sprintf("a tab of kind %q is in front of them", t.Kind)
	}
	return presenter.describe(t, formatTabName(t), tabPosition(t))
}

// formatTabName returns how a tab is named in a sentence.
func formatTabName(t domain.Tab) string {
	switch {
	case t.Title != "" && t.Path != "":
		return fmt.Sprintf("%q at %s", t.Title, t.Path)
	case t.Path != "":
		return t.Path
	}
	return fmt.Sprintf("%q", t.Title)
}

// describeWithPosition formats a tab description that includes its position.
func describeWithPosition(what string) func(t domain.Tab, name, where string) string {
	return func(_ domain.Tab, name, where string) string {
		if where == "" {
			return fmt.Sprintf("the %s %s is in front of them", what, name)
		}
		return fmt.Sprintf("the %s %s is in front of them, open at %s", what, name, where)
	}
}

// documentPosition returns the page of the document.
func documentPosition(t domain.Tab) string {
	if t.Document == nil || t.Document.PageCount <= 0 {
		return ""
	}
	return fmt.Sprintf("page %d of %d", t.Document.Page, t.Document.PageCount)
}

// bookPosition returns where the person is in a book.
func bookPosition(t domain.Tab) string {
	if t.Book == nil || t.Book.PageCount <= 0 {
		return ""
	}
	return fmt.Sprintf("page %d of %d, at byte %d", t.Book.Page, t.Book.PageCount, t.Book.Offset)
}

// recordingPosition returns how much of the recording is transcribed.
func recordingPosition(t domain.Tab) string {
	var writtenTo, length int
	if t.Recording != nil {
		writtenTo, length = t.Recording.TranscribedDuration, t.Recording.Duration
	}
	switch {
	case writtenTo <= 0:
		return "none of it written down yet"
	case length <= 0:
		return fmt.Sprintf("%s of it written down", transcript.Clock(writtenTo))
	}
	return fmt.Sprintf("%s of its %s written down", transcript.Clock(writtenTo), transcript.Clock(length))
}
