// Package api holds the arguments and results of the window tools.
package api

// Place is a place in a file: a page of a PDF, a position in a recording, or a position in an EPUB.
type Place struct {
	Page            int      `json:"page,omitempty" jsonschema:"PDF: the page shown, from 1"`
	PageCount       int      `json:"pageCount,omitempty" jsonschema:"PDF: the number of pages"`
	PositionSeconds *float64 `json:"positionSeconds,omitempty" jsonschema:"recording: seconds from the start"`
	CFI             string   `json:"cfi,omitempty" jsonschema:"EPUB: the CFI of the place shown"`
	Chapter         string   `json:"chapter,omitempty" jsonschema:"EPUB: the table-of-contents title at the place"`
}

// TabInfo describes an open tab in the user window.
type TabInfo struct {
	TabID    string `json:"tabId" jsonschema:"the identifier of the tab"`
	Kind     string `json:"kind" jsonschema:"the kind of the tab (e.g. note, document, book, recording, plex)"`
	Path     string `json:"path,omitempty" jsonschema:"the path of the file in the tab, relative to the vault folder"`
	Title    string `json:"title,omitempty" jsonschema:"the tab title"`
	Place    *Place `json:"place,omitempty" jsonschema:"the place the tab is open at, omitted for a tab with none"`
	IsActive bool   `json:"isActive,omitempty" jsonschema:"true for the active tab"`
}

// WindowTabListInput represents the arguments for window_tab_list.
type WindowTabListInput struct{}

// WindowTabListOutput represents the result of inspecting open window tabs.
type WindowTabListOutput struct {
	Tabs        []TabInfo `json:"tabs" jsonschema:"the open tabs, in window order"`
	ActiveTabID string    `json:"activeTabId,omitempty" jsonschema:"the identifier of the active tab"`
}

// WindowTabActivateInput represents the arguments for window_tab_activate.
type WindowTabActivateInput struct {
	TabID string `json:"tabId" jsonschema:"the identifier of the tab to make active, as window_tab_list returns it"`
}

// WindowTabActivateOutput represents the result of activating a tab.
type WindowTabActivateOutput struct {
	Tab TabInfo `json:"tab" jsonschema:"the active tab"`
}

// NoteEditPreviewInput represents the arguments for note_edit_preview.
type NoteEditPreviewInput struct {
	Path         string `json:"path" jsonschema:"the note to propose changes to, by its path relative to the vault folder"`
	ProposedText string `json:"proposedText,omitempty" jsonschema:"the proposed new text of the note"`
	OriginalText string `json:"originalText,omitempty" jsonschema:"optional baseline text to diff against if not reading from vault"`
	Diff         string `json:"diff,omitempty" jsonschema:"optional unified diff patch to preview directly"`
}

// NoteEditPreviewOutput represents the result of generating an edit preview.
type NoteEditPreviewOutput struct {
	PreviewID string `json:"previewId" jsonschema:"unique identifier for the proposed edit preview"`
	Path      string `json:"path" jsonschema:"the path of the note, relative to the vault folder"`
	Diff      string `json:"diff" jsonschema:"unified diff representation of the proposed changes"`
	Additions int    `json:"additions" jsonschema:"number of lines added"`
	Deletions int    `json:"deletions" jsonschema:"number of lines removed"`
}

// NoteFocusInput represents the arguments for note_focus.
type NoteFocusInput struct {
	Path string `json:"path" jsonschema:"the note to focus, by its path relative to the vault folder"`
}

// FocusedNote is the note note_focus made the current one.
type FocusedNote struct {
	Path  string `json:"path" jsonschema:"the path of the note, relative to the vault folder"`
	Title string `json:"title" jsonschema:"the title of the note"`
	ID    string `json:"id,omitempty" jsonschema:"the stable identifier of the note, empty for a note written outside the application"`
}

// NoteFocusOutput represents the result of note_focus.
type NoteFocusOutput struct {
	Note FocusedNote `json:"note" jsonschema:"the focused note"`
}

// Span is a run of a file's text version in bytes, as a search of the library returns it.
type Span struct {
	Start  int `json:"start" jsonschema:"the byte offset of the passage in the text of the file, as a search of the library returns it"`
	Length int `json:"length" jsonschema:"the length of the passage in bytes, as a search of the library returns it"`
}

// SourceFocusInput represents the arguments for source_focus.
type SourceFocusInput struct {
	Path  string `json:"path" jsonschema:"the file to open, by its path relative to the vault folder"`
	Spans []Span `json:"spans,omitempty" jsonschema:"the passages of the file to highlight; the window scrolls to the first, and an empty list opens the file at its beginning"`
}

// SourceFocusOutput represents the result of source_focus.
type SourceFocusOutput struct {
	Path           string `json:"path" jsonschema:"the path of the opened file, relative to the vault folder"`
	HighlightCount int    `json:"highlightCount" jsonschema:"the number of highlighted passages; 0 for a note, which opens whole"`
}
