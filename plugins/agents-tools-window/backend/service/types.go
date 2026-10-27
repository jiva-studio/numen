// Package service implements the Window Interaction & Edit Preview Tools plugin backend.
package service

import "time"

// TabInfo describes an open tab in the user window.
type TabInfo struct {
	ID       string `json:"id" jsonschema:"the identifier of the tab"`
	Kind     string `json:"kind" jsonschema:"what kind of tab it is (e.g. note, document, book, recording, plex)"`
	Path     string `json:"path,omitempty" jsonschema:"the file it holds, by the path the vault files it under"`
	Title    string `json:"title,omitempty" jsonschema:"what the tab is called, as the person reads it"`
	Where    string `json:"where,omitempty" jsonschema:"where in that file the person stands"`
	IsActive bool   `json:"active,omitempty" jsonschema:"whether this tab is the active one"`
}

// WindowTabsGetInput represents the arguments for window_tabs_get.
type WindowTabsGetInput struct{}

// WindowTabsGetOutput represents the result of inspecting open window tabs.
type WindowTabsGetOutput struct {
	Tabs     []TabInfo `json:"tabs"`
	ActiveID string    `json:"activeId,omitempty" jsonschema:"the identifier of the active tab"`
	Summary  string    `json:"summary" jsonschema:"the tab the person is looking at, in words to say back to them"`
}

// WindowFocusTabInput represents the arguments for window_focus_tab.
type WindowFocusTabInput struct {
	TabID string `json:"tabId,omitempty" jsonschema:"the identifier of the tab to focus"`
	Path  string `json:"path,omitempty" jsonschema:"the path of the note or file to focus"`
	Title string `json:"title,omitempty" jsonschema:"the title of the tab to focus"`
}

// WindowFocusTabOutput represents the result of focusing a tab.
type WindowFocusTabOutput struct {
	Success bool    `json:"success"`
	Tab     TabInfo `json:"tab"`
	Message string  `json:"message"`
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
	Path      string `json:"path"`
	Diff      string `json:"diff" jsonschema:"unified diff representation of the proposed changes"`
	Additions int    `json:"additions" jsonschema:"number of lines added"`
	Deletions int    `json:"deletions" jsonschema:"number of lines removed"`
	Status    string `json:"status"`
	Summary   string `json:"summary"`
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
