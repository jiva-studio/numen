package service

import (
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/usecase/arrange"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/usecase/library"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/usecase/links"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/usecase/vaults"
)

func createLibraryTools() []registration {
	return []registration{
		bind(requirePermission(createReadSpec("library_search", "Search the library",
			"Search the text index of all files in the vault: the notes, and the books, "+
				"PDFs, scans and recordings stored with them, by the words of the query and by their meaning. "+
				"Returns passages, best first: the text around each hit, the path of the file, its place "+
				"(a page, a chapter, a time) and its address. One file returns at most three passages. "+
				"note_search searches only notes, by title, path and text; use library_search for the "+
				"text of a book, a PDF or the transcript of a recording. "+
				"A passage is truncated to a fixed size, often mid-sentence: read more with library_read "+
				"before concluding that a source does not cover a subject. For a question about a book, "+
				"pass kinds [\"file\"]: a vault has far more notes than books, and a search of every "+
				"kind returns mostly notes.",
			noteLinkSentence, passageLinkSentence, noFileLinkSentence, searchScopeSentence, userWordsSentence, truncatedSentence, bookKindSentence),
			host.PermissionIndexRead),
			(*call).createLibrary, (*library.UseCase).Search),
		bind(requirePermission(createReadSpec("library_read", "Read the text of a file",
			"Read the text of one file of the vault from its text version: the pages of a PDF, "+
				"the chapters of a book, the transcript of a recording, the recognized text of a scan. "+
				"Give a page, or a start and a length in bytes; the result lists the ranges of the text with "+
				"their places in the source, the length of the whole text, and the start to read next. "+
				"A text file is read as it is. A file with no text version returns an error: it must be "+
				"recognized or transcribed first. file_read reads the bytes of a text file and returns an error "+
				"for a binary file; "+
				"library_search finds the passage to read.",
			passageLinkSentence, truncatedSentence, libraryReadSentence),
			host.PermissionIndexRead),
			(*call).createLibrary, (*library.UseCase).Read),
	}
}

func createInspectTools() []registration {
	return []registration{
		bind(createReadSpec("link_list", "List Links",
			"List outgoing wikilinks and incoming backlinks connected to a note."),
			(*call).createLinks, (*links.UseCase).List),
		bind(createReadSpec("vault_get", "Get Vault Info",
			"Get the identifier, name, root path and note count of a vault, named by its name, path or identifier; the vault of the call when none is named."),
			(*call).createVaults, (*vaults.UseCase).Get),
		bind(createReadSpec("vault_list", "List Vaults",
			"List every vault of the installation with its identifier, name and folder path. The current "+
				"vault is marked isCurrent, and a vault whose folder is missing is marked isMissing.",
			vaultListSentence),
			(*call).createVaults, (*vaults.UseCase).List),
		bind(createReadSpec("vault_problem_list", "List Vault Problems",
			"Run checks over the vault and list the problems found: parse for files that could not be "+
				"read, frontmatter for frontmatter that is not valid, ambiguous for links that resolve to several "+
				"notes, and dangling for links that resolve to no note. parse, frontmatter and ambiguous run by "+
				"default; dangling runs only when named.",
			vaultProblemListSentence),
			(*call).createArrange, (*arrange.UseCase).ListProblems),
	}
}

func createArrangeTools() []registration {
	return []registration{
		bind(createWriteSpec("note_rename", "Rename Note",
			"Rename a note. The title is written in the title key when the note has one, "+
				"else in the file name, and the file moves to the file name of the title. The host rewrites "+
				"the links that use the old name of the note.",
			noteRenameSentence, linkRewriteSentence),
			(*call).createArrange, (*arrange.UseCase).Rename),
		bind(createWriteSpec("note_move", "Move Notes",
			"Move up to 50 notes to a folder, keeping their file names; the folder \"/\" is the vault "+
				"root. Each path gets its own result, and a path that fails does not stop the others.",
			noteMoveSentence, linkRewriteSentence),
			(*call).createArrange, (*arrange.UseCase).Move),
		bind(createWriteSpec("note_remove", "Remove Notes",
			"Move up to 50 notes to the vault's trash, from which they can be restored. A folder is "+
				"rejected. Each path gets its own result with the notes whose links are now broken.",
			noteRemoveSentence),
			(*call).createArrange, (*arrange.UseCase).Remove),
	}
}

func createVaultListTools() []registration {
	return []registration{
		bind(requireManagePermission(createWriteSpec("vault_add", "Add Vault",
			"Add a folder of this machine to the installation's list of vaults, by its absolute path. "+
				"A folder that is inside a vault on the list, or that contains one, is rejected.",
			vaultAddSentence)),
			(*call).createVaults, (*vaults.UseCase).Add),
		bind(requireManagePermission(createWriteSpec("vault_rename", "Rename Vault",
			"Change the display name of a vault, named by its name, path or identifier. A name "+
				"another vault has is rejected.",
			vaultRenameSentence)),
			(*call).createVaults, (*vaults.UseCase).Rename),
		bind(requireManagePermission(createWriteSpec("vault_remove", "Remove Vault",
			"Remove a vault from the installation's list; its folder is not deleted. The current "+
				"vault and the last vault are rejected.",
			vaultRemoveSentence)),
			(*call).createVaults, (*vaults.UseCase).Remove),
		bind(requireManagePermission(createWriteSpec("vault_open", "Open Vault",
			"Switch the window to another vault, named by its name, path or identifier, and return "+
				"when the window shows it; this session then loses access to the current vault. If the window "+
				"has unsaved changes, the vault is not switched, the window prompts the user, and the call returns "+
				"an error; call it again after the user responds. Call it only when the user asks for it.",
			vaultOpenSentence, vaultOpenPromptSentence)),
			(*call).createVaults, (*vaults.UseCase).Open),
	}
}
