package hintsource

import "github.com/jiva-studio/numen/libs/plugin-catalog/manifest"

const (
	categoryViews   = "views"
	categoryReaders = "readers"
	iconFileText    = "file-text"
)

// BuiltIn returns the hints of the plugins that ship with the application. Each repeats the
// identity and the contributions of the manifest of the plugin it names.
func BuiltIn() []Hint {
	return append(listFileHints(), listViewHints()...)
}

// newHint returns the hint of a plugin that contributes the tab kinds and opens the extensions.
func newHint(id, name, description, feature, category string, kinds []manifest.TabKind, extensions ...string) Hint {
	return Hint{
		ID:          id,
		Name:        name,
		Description: description,
		Contributes: manifest.Contributes{TabKinds: kinds, Feature: feature, Category: category},
		Extensions:  extensions,
		FileNames:   nil,
	}
}

// listFileHints are the plugins that open files.
func listFileHints() []Hint {
	return []Hint{
		newHint("md.numen.notes", "Notes", "Markdown note editor with live formatting and wikilinks",
			"Markdown notes", "editors",
			[]manifest.TabKind{
				{Kind: "note", Title: "Note", Icon: iconFileText},
				{Kind: "reader:markdown", Title: "Markdown Note Editor", Icon: iconFileText},
			},
			"md", "markdown"),
		newHint("md.numen.readers.epub", "EPUB Book Reader", "EPUB book reader with highlights and passage links",
			"EPUB books", categoryReaders,
			[]manifest.TabKind{{Kind: "reader:reader-epub", Title: "EPUB Reader", Icon: "book-open"}},
			"epub"),
		newHint("md.numen.readers.pdf", "PDF Reader", "PDF document reader with highlights and passage links",
			"PDF documents", categoryReaders,
			[]manifest.TabKind{{Kind: "reader:reader-pdf", Title: "PDF Reader", Icon: iconFileText}},
			"pdf"),
		newHint("md.numen.views.media.player", "Audio Player & Recorder",
			"Audio and video player with interactive transcripts and voice recording",
			"audio playback and recording", categoryViews,
			[]manifest.TabKind{
				{Kind: "recording", Title: "Record Audio", Icon: "mic"},
				{Kind: "reader:media-player", Title: "Audio Player", Icon: "audio-lines"},
			},
			"mp3", "m4a", "wav", "ogg", "oga", "aac", "flac"),
	}
}

// listViewHints are the plugins that open tabs of their own and no file.
func listViewHints() []Hint {
	return []Hint{
		newHint("md.numen.tools.media.importer", "Media Importer",
			"Saves web articles and downloads online videos for offline playback",
			"media links", "tools",
			[]manifest.TabKind{{Kind: "link", Title: "Link", Icon: "link"}}),
		newHint("md.numen.views.plex.graph", "Plex Graph", "Interactive visual map of connected notes and topics",
			"plex graph", categoryViews,
			[]manifest.TabKind{{Kind: "plex", Title: "Plex", Icon: "waypoints"}}),
		newHint("md.numen.views.agent.chat", "Agent Chat", "Interactive chat tab for working with AI assistants",
			"agent chat", categoryViews,
			[]manifest.TabKind{{Kind: "agent", Title: "Agent", Icon: "bot"}}),
		newHint("md.numen.views.file.tree", "File Tree Explorer",
			"Sidebar file browser for managing notes, books, and recordings",
			"file tree", categoryViews,
			[]manifest.TabKind{{Kind: "files", Title: "Files", Icon: "folder-tree"}}),
		newHint("md.numen.views.flashcards.studio", "Flashcards Studio",
			"Flashcard deck editor, card template designer, and spaced-repetition review",
			"flashcards", categoryViews,
			[]manifest.TabKind{
				{Kind: "deck", Title: "Deck", Icon: "layers"},
				{Kind: "card-model", Title: "Card Model", Icon: "layout-template"},
				{Kind: "preset", Title: "Preset", Icon: "gauge"},
				{Kind: "flashcards.review", Title: "Flashcards", Icon: ""},
			}),
	}
}
