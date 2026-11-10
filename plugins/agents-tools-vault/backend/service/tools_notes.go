package service

import (
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/usecase/files"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/usecase/links"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/usecase/notes"
)

func noteTools() []registration {
	return []registration{
		bind(reading(tools.ByEffect, "note_read", "Read Note",
			"Read the content, body prose, title, and metadata of a note in the vault."),
			(*call).notes, (*notes.UseCase).Read),
		bind(writing(tools.ByEffect, "note_write", "Write Note",
			"Atomically write or create a note in the vault. If fingerprint is provided, verifies optimistic concurrency."),
			(*call).notes, (*notes.UseCase).Write),
		bind(reading(tools.ByEffect, "note_heading_list", "Extract Note Headings",
			"Extract the outline and headings from a note in the vault."),
			(*call).notes, (*notes.UseCase).ListHeadings),
		bind(reading(tools.Search, "note_search", "Search Notes",
			"Search the notes of the vault by title, path and text. It does not search books, "+
				"PDFs, scans or recordings: library_search finds the text of those.",
			noteLinkSentence, passageLinkSentence, noFileLinkSentence, searchScopeSentence, userWordsSentence),
			(*call).notes, (*notes.UseCase).Search),
		bind(reading(tools.ByEffect, "note_list", "List Notes",
			"List the paths and titles of the notes in the vault or in one folder, ordered by path. The result is paged with limit and offset and gives the total count.",
			noteListSentence),
			(*call).notes, (*notes.UseCase).List),
		bind(reading(tools.ByEffect, "note_title_list", "Get Note Titles",
			"Get the titles and identifiers of up to 50 notes by their paths. Paths that name no note are returned in missingPaths.",
			noteTitleListSentence),
			(*call).notes, (*notes.UseCase).ListTitles),
		bind(reading(tools.ByEffect, "note_resolve", "Resolve Note Name",
			"Get the paths of every note with the given file name, which is what a wikilink names. More than one path means the name is ambiguous.",
			noteResolveSentence),
			(*call).notes, (*notes.UseCase).ResolveName),
	}
}

func fileTools() []registration {
	return []registration{
		bind(reading(tools.ByEffect, "file_read", "Read File",
			"Read the content or a byte range of a text file of the vault, as text. A file that is "+
				"not text — a PDF, a book, a recording — returns an error; library_read reads its text.",
			fileReadSentence),
			(*call).files, (*files.UseCase).Read),
		bind(writing(tools.ByEffect, "file_create", "Create File",
			"Create a new raw file in the vault with the specified content."),
			(*call).files, (*files.UseCase).Create),
	}
}

func linkEditTools() []registration {
	return []registration{
		bind(writing(tools.ByEffect, "link_add", "Add Links",
			"Write up to 50 links into the frontmatter links of notes. Each link needs the fingerprint "+
				"note_read returned for its note; the links for one note are written together and need the same "+
				"fingerprint. With a stale fingerprint, no link of that note is written and the note is unchanged.",
			linkEditSentence),
			(*call).links, (*links.UseCase).Add),
		bind(writing(tools.ByEffect, "link_update", "Update Link",
			"Change the role, type, label or note of the links a note has to a target, keeping the "+
				"target. Fields left out are kept. Needs the fingerprint note_read returned "+
				"for the note; a stale one is rejected.",
			linkEditSentence),
			(*call).links, (*links.UseCase).Update),
		bind(writing(tools.Remove, "link_remove", "Remove Links",
			"Remove the links a note has to a target from its frontmatter, or only the link of one "+
				"role. The note at the other end is unchanged. Needs the fingerprint note_read returned for the note, "+
				"and a stale one is rejected.",
			linkEditSentence),
			(*call).links, (*links.UseCase).Remove),
	}
}
