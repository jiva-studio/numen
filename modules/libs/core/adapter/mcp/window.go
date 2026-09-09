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
				Where: stands(one),
				Front: front,
			})
			if front {
				res.Looking = inFront(one)
			}
		}
		return nil, res, nil
	})
}

// spoken is what a tab of one kind says: where in it the person stands, and
// how the one in front of them is said back. The words are this adapter's —
// domain says what a tab holds, and this says what that is to a person. A
// kind nothing here has heard of is named by its own word and nothing more.
type spoken struct {
	// standing is where in the tab the person stands. A tab that measures
	// nothing leaves it unset.
	standing func(t domain.Tab) string
	// inFront is the tab said back to the person, by the name they call it by
	// and where they stand in it.
	inFront func(t domain.Tab, name, where string) string
}

// spokenBy is what each kind of tab says, one entry to a kind. A new kind of
// tab is one entry here.
var spokenBy = map[string]spoken{
	domain.TabNote: {
		inFront: func(t domain.Tab, name, _ string) string {
			return fmt.Sprintf("the note %s is in front of them", name)
		},
	},
	domain.TabPlex: {
		inFront: func(t domain.Tab, name, _ string) string {
			if t.Path == "" {
				return "they are looking at a plex standing on no note"
			}
			return fmt.Sprintf("they are looking at the plex around the note %s", name)
		},
	},
	domain.TabDocument: {
		standing: documentStands,
		inFront:  whereFront("document"),
	},
	domain.TabBook: {
		standing: bookStands,
		inFront:  whereFront("book"),
	},
	domain.TabRecording: {
		standing: recordingStands,
		inFront: func(t domain.Tab, name, where string) string {
			return fmt.Sprintf("the recording %s is in front of them, with %s", name, where)
		},
	},
}

// stands is where in what a tab holds the person stands, in the terms that tab
// measures in. A tab that measures nothing says nothing.
func stands(t domain.Tab) string {
	words, known := spokenBy[t.Kind]
	if !known || words.standing == nil {
		return ""
	}
	return words.standing(t)
}

// inFront is the tab the person is looking at, said back to them. A kind this
// application has no words for is named by its own word and nothing more.
func inFront(t domain.Tab) string {
	words, known := spokenBy[t.Kind]
	if !known {
		return fmt.Sprintf("a tab of kind %q is in front of them", t.Kind)
	}
	return words.inFront(t, called(t), stands(t))
}

// called is how a tab is named in a sentence: what the person calls it, and
// the file it holds so that a tool can be asked about it.
func called(t domain.Tab) string {
	switch {
	case t.Title != "" && t.Path != "":
		return fmt.Sprintf("%q at %s", t.Title, t.Path)
	case t.Path != "":
		return t.Path
	}
	return fmt.Sprintf("%q", t.Title)
}

// whereFront is how a tab that is open at a place in it is said back: the
// thing is in front of them, and the place is named when the tab stands at one.
func whereFront(what string) func(t domain.Tab, name, where string) string {
	return func(_ domain.Tab, name, where string) string {
		if where == "" {
			return fmt.Sprintf("the %s %s is in front of them", what, name)
		}
		return fmt.Sprintf("the %s %s is in front of them, open at %s", what, name, where)
	}
}

// documentStands is the page of the document the person stands on.
func documentStands(t domain.Tab) string {
	if t.Document == nil || t.Document.PageCount <= 0 {
		return ""
	}
	return fmt.Sprintf("page %d of %d", t.Document.Page, t.Document.PageCount)
}

// bookStands is where the person stands in a book.
func bookStands(t domain.Tab) string {
	// A place in a book that reflows is an offset, and the page is how far
	// through that offset stands.
	if t.Book == nil || t.Book.PageCount <= 0 {
		return ""
	}
	return fmt.Sprintf("page %d of %d, at byte %d", t.Book.Page, t.Book.PageCount, t.Book.Offset)
}

// recordingStands is how much of the recording is written down.
func recordingStands(t domain.Tab) string {
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
