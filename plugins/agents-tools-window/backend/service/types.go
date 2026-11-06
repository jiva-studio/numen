// Package service implements the Window Interaction & Edit Preview Tools plugin backend.
package service

import "time"

// TabInfo describes an open tab in the user window.
type TabInfo struct {
	ID       string `json:"id" jsonschema:"the identifier of the tab"`
	Kind     string `json:"kind" jsonschema:"the kind of the tab (e.g. note, document, book, recording, plex)"`
	Path     string `json:"path,omitempty" jsonschema:"the path of the file in the tab, relative to the vault folder"`
	Title    string `json:"title,omitempty" jsonschema:"the tab title"`
	Location string `json:"location,omitempty" jsonschema:"the location in the file, as the window reports it (e.g. page 4 of 120)"`
	IsActive bool   `json:"isActive,omitempty" jsonschema:"true for the active tab"`
}

// WindowTabsGetInput represents the arguments for window_tabs_get.
type WindowTabsGetInput struct{}

// WindowTabsGetOutput represents the result of inspecting open window tabs.
type WindowTabsGetOutput struct {
	Tabs     []TabInfo `json:"tabs" jsonschema:"the open tabs, in window order"`
	ActiveID string    `json:"activeId,omitempty" jsonschema:"the identifier of the active tab"`
}

// WindowFocusTabInput represents the arguments for window_focus_tab.
type WindowFocusTabInput struct {
	TabID string `json:"tabId,omitempty" jsonschema:"the identifier of the tab to focus"`
	Path  string `json:"path,omitempty" jsonschema:"the path of the note or file to focus"`
	Title string `json:"title,omitempty" jsonschema:"the title of the tab to focus"`
}

// WindowFocusTabOutput represents the result of focusing a tab.
type WindowFocusTabOutput struct {
	Tab TabInfo `json:"tab" jsonschema:"the focused tab"`
}

// EditPreviewInput represents the arguments for edit_preview.
type EditPreviewInput struct {
	Path            string `json:"path" jsonschema:"the note to propose changes to, by its path relative to the vault folder"`
	ProposedContent string `json:"proposedContent,omitempty" jsonschema:"the proposed new content of the note"`
	OriginalContent string `json:"originalContent,omitempty" jsonschema:"optional baseline content to diff against if not reading from vault"`
	Diff            string `json:"diff,omitempty" jsonschema:"optional unified diff patch to preview directly"`
}

// EditPreviewOutput represents the result of generating an edit preview.
type EditPreviewOutput struct {
	PreviewID string `json:"previewId" jsonschema:"unique identifier for the proposed edit preview"`
	Path      string `json:"path" jsonschema:"the path of the note, relative to the vault folder"`
	Diff      string `json:"diff" jsonschema:"unified diff representation of the proposed changes"`
	Additions int    `json:"additions" jsonschema:"number of lines added"`
	Deletions int    `json:"deletions" jsonschema:"number of lines removed"`
}

// EditPreview stores a pending or proposed edit preview.
type EditPreview struct {
	ID              string
	Path            string
	ProposedContent string
	OriginalContent string
	Diff            string
	Additions       int
	Deletions       int
	CreatedAt       time.Time
}

// NoteFocusInput represents the arguments for note_focus.
type NoteFocusInput struct {
	Path string `json:"path" jsonschema:"the note to put in focus, by its path relative to the vault folder"`
}

// FocusedNote is the note note_focus made the current one.
type FocusedNote struct {
	Path  string `json:"path" jsonschema:"the path of the note, relative to the vault folder"`
	Title string `json:"title" jsonschema:"the title of the note"`
	ID    string `json:"id,omitempty" jsonschema:"the stable identifier of the note, empty for a note written outside the application"`
}

// NoteFocusOutput represents the result of note_focus.
type NoteFocusOutput struct {
	Note FocusedNote `json:"note" jsonschema:"the note now in focus"`
}

// Passage is a run of a file's text version in bytes, as library_search returns it.
type Passage struct {
	Start  int `json:"start" jsonschema:"the byte offset of the passage in the text of the file, as library_search returns it"`
	Length int `json:"length" jsonschema:"the length of the passage in bytes, as library_search returns it"`
}

// SourceFocusInput represents the arguments for source_focus.
type SourceFocusInput struct {
	Path     string    `json:"path" jsonschema:"the file to open, by its path relative to the vault folder"`
	Passages []Passage `json:"passages,omitempty" jsonschema:"the passages of the file to highlight; the window scrolls to the first, and an empty list opens the file at its beginning"`
}

// SourceFocusOutput represents the result of source_focus.
type SourceFocusOutput struct {
	Path           string `json:"path" jsonschema:"the path of the opened file, relative to the vault folder"`
	HighlightCount int    `json:"highlightCount" jsonschema:"the number of highlighted passages"`
}
