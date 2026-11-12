package service

import (
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/usecase/files"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/usecase/links"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/usecase/notes"
)

func createNoteTools() []registration {
	return []registration{
		bind(createReadSpec("note_read", "Read Note",
			"Read the content, body prose, title, and metadata of a note in the vault."),
			(*call).createNotes, (*notes.UseCase).Read),
		bind(createWriteSpec("note_write", "Write Note",
			"Atomically write or create a note in the vault. If fingerprint is provided, verifies optimistic concurrency."),
			(*call).createNotes, (*notes.UseCase).Write),
		bind(createReadSpec("note_heading_list", "Extract Note Headings",
			"Extract the outline and headings from a note in the vault."),
			(*call).createNotes, (*notes.UseCase).ListHeadings),
		bind(createReadSpec("note_search", "Search Notes",
			"Search the notes of the vault by title, path and text. It does not search books, "+
				"PDFs, scans or recordings: library_search finds the text of those.",
			noteLinkSentence, passageLinkSentence, noFileLinkSentence, searchScopeSentence, userWordsSentence),
			(*call).createNotes, (*notes.UseCase).Search),
		bind(createReadSpec("note_list", "List Notes",
			"List the paths and titles of the notes in the vault or in one folder, ordered by path. The result is paged with limit and offset and gives the total count.",
			noteListSentence),
			(*call).createNotes, (*notes.UseCase).List),
		bind(createReadSpec("note_title_list", "Get Note Titles",
			"Get the titles and identifiers of up to 50 notes by their paths. Paths that name no note are returned in missingPaths.",
			noteTitleListSentence),
			(*call).createNotes, (*notes.UseCase).ListTitles),
		bind(createReadSpec("note_resolve", "Resolve Note Name",
			"Get the paths of every note with the given file name, which is what a wikilink names. More than one path means the name is ambiguous.",
			noteResolveSentence),
			(*call).createNotes, (*notes.UseCase).ResolveName),
	}
}

func createFileTools() []registration {
	return []registration{
		bind(createReadSpec("file_read", "Read File",
			"Read the content or a byte range of a text file of the vault, as text. A file that is "+
				"not text — a PDF, a book, a recording — returns an error; library_read reads its text.",
			fileReadSentence),
			(*call).createFiles, (*files.UseCase).Read),
		bind(createWriteSpec("file_create", "Create File",
			"Create a new raw file in the vault with the specified content."),
			(*call).createFiles, (*files.UseCase).Create),
	}
}

func createLinkEditTools() []registration {
	return []registration{
		bind(createWriteSpec("link_add", "Add Links",
			"Write up to 50 links into the frontmatter links of notes. Each link needs the fingerprint "+
				"note_read returned for its note; the links for one note are written together and need the same "+
				"fingerprint. With a stale fingerprint, no link of that note is written and the note is unchanged.",
			linkEditSentence),
			(*call).createLinks, (*links.UseCase).Add),
		bind(createWriteSpec("link_update", "Update Link",
			"Change the role, type, label or note of the links a note has to a target, keeping the "+
				"target. Fields left out are kept. Needs the fingerprint note_read returned "+
				"for the note; a stale one is rejected.",
			linkEditSentence),
			(*call).createLinks, (*links.UseCase).Update),
		bind(createWriteSpec("link_remove", "Remove Links",
			"Remove the links a note has to a target from its frontmatter, or only the link of one "+
				"role. The note at the other end is unchanged. Needs the fingerprint note_read returned for the note, "+
				"and a stale one is rejected.",
			linkEditSentence),
			(*call).createLinks, (*links.UseCase).Remove),
	}
}
