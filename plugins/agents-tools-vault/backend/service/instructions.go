package service

// The instruction sentences of the vault tools. A sentence about several tools is attached to each
// of them, and the host lists it once.
const (
	noteLinkSentence = "Link each note you mention by its title where you mention it, for example [[Tidal pools]], " +
		"not in a list at the end; when several notes share a title, link the note as [[note://<identifier>]]."
	resultLinkSentence  = "Link each passage of a book, a PDF or a recording that you cite where you cite it, as a markdown link to the `uri` that library_search returns for it, not in a list at the end, for example [the harbor chapter](numen:library%2FA%20Sea%20Atlas.pdf?start=40120&length=980)."
	noFileLinkSentence  = "Never use file:// links for notes or books."
	searchScopeSentence = "note_search finds notes only; library_search finds passages in all indexed files: notes, books, PDFs, scans and recordings."
	userWordsSentence   = "Search with the user's own words before your own."
	truncatedSentence   = "A passage is truncated, often mid-sentence: read more with library_read before you conclude that a source does not cover a topic."
	bookKindSentence    = "For a question about a book, pass kinds [\"file\"] to library_search, because a vault has far more notes than books."
	libraryReadSentence = "library_read reads the text of one source a page or a range at a time."
	fileReadSentence    = "file_read reads text files and returns an error for a binary file."

	noteListSentence         = "Use note_list to list the notes of the vault or of one folder by path and title, and page through them with limit and offset."
	noteTitleListSentence    = "Use note_title_list to get the titles and identifiers of notes whose paths you already have."
	noteResolveSentence      = "Use note_resolve to get the paths of the notes a wikilink name points to; more than one path means the name is ambiguous."
	vaultListSentence        = "vault_list lists every vault of the installation, marks the current vault with isCurrent and a vault whose folder is missing with isMissing."
	vaultProblemListSentence = "vault_problem_list lists the problems its checks find in the vault, and runs the dangling check only when you name it."
	noteRenameSentence       = "note_rename renames a note and moves its file to the file name of the new title."
	noteMoveSentence         = "note_move moves notes to a folder."
	noteRemoveSentence       = "note_remove moves notes to the vault's trash, from which they can be restored."
	linkRewriteSentence      = "After note_rename or note_move the host rewrites the links that use the old name of the note."
	linkEditSentence         = "link_add, link_update and link_remove write the links in the frontmatter of a note and need the fingerprint note_read returned for that note; a stale fingerprint is rejected, so read the note again."
	vaultAddSentence         = "vault_add adds a folder as a vault."
	vaultRenameSentence      = "vault_rename changes the name of a vault."
	vaultRemoveSentence      = "vault_remove removes a vault from the list and does not delete its folder."
	vaultOpenSentence        = "vault_open switches the user's window to another vault and ends this session's access to the current vault, so call it only when the user asks for it."
	vaultOpenPromptSentence  = "When the window has unsaved changes, vault_open does not switch the vault, the window prompts the user, and the call returns an error; call it again after the user responds."
)
